package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	elogrus "github.com/dictor/echologrus"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

var (
	// GlobalLogger is global logger instance
	GlobalLogger *logrus.Logger
	// AllowedExtension contains allowed extensions for video scan in getVideoPaths function
	AllowedExtension string
	// ThumbnailDir is directory for making and reading thumbnail
	ThumbnailDir string
	// VideoRootDir is root directory of videos
	VideoRootDir string
	// ThumbnailMinimumInterval is minimum frame interval second of animated thumbnail
	ThumbnailMinimumInterval int
)

type (
	// Video contains each video's detail
	Video struct {
		Path       string    `json:"path"`
		Hash       string    `json:"hash"`
		Name       string    `json:"name"`
		Size       int64     `json:"size"`
		ModifiedAt time.Time `json:"modified_at"`
	}

	// NameFunction make video id (like hash) from path
	NameFunction func(string) (string, error)
)

func main() {
	e := echo.New()
	GlobalLogger = elogrus.Attach(e).Logger

	// Parse flag options
	flag.StringVar(&AllowedExtension, "ext", ".mkv .mp4 .webm .avi", "allowed video file extension")
	flag.StringVar(&ThumbnailDir, "tdir", "", "directory for making and reading thumbnail. When empty, make 'thumb' directory on binary's directory and use it (it must be absolute path!)")
	flag.StringVar(&VideoRootDir, "vdir", "", "(required) root directory of videos")
	flag.IntVar(&ThumbnailMinimumInterval, "tint", 200, "minimum frame interval second of animated thumbnail")
	flag.Parse()

	if VideoRootDir == "" {
		GlobalLogger.Fatal("video root directory isn't given!")
	}

	// Checking ffmpeg is existing
	if err := checkFFmpeg(); err != nil {
		GlobalLogger.WithError(err).Fatal("ffmpeg check error")
	}

	// Scan video file
	videos, err := getVideos(VideoRootDir, getFileBase64)
	if err != nil {
		GlobalLogger.WithError(err).Fatal("video scan error")
	}

	// Start thumbnail making task
	thumbDir, err := getThumbnailDirectory()
	if err != nil {
		GlobalLogger.WithError(err).Fatal("thumbnail directory retrieve error")
	}
	go startThumbnailTask(videos, thumbDir)

	e.Static("/thumb", thumbDir)
	e.File("/", "static/index.html")
	e.File("/script", "static/script.js")
	e.File("/style", "static/style.css")
	e.File("/v-lazy-image", "static/v-lazy-image.js")
	e.GET("/video", func(c echo.Context) error {
		return c.JSON(http.StatusOK, videos)
	})
	e.Logger.Fatal(e.Start(":80"))
}

func startThumbnailTask(videos []Video, thumbDir string) {
	workingDir, err := ioutil.TempDir("", "thumbnailer")
	if err != nil {
		GlobalLogger.Fatal(err)
	}
	defer os.RemoveAll(workingDir)

	GlobalLogger.WithFields(logrus.Fields{"thumbnail_dir": thumbDir, "temp_dir": workingDir}).Info("thumbnail task start")

	for _, video := range videos {
		if _, err := os.Stat(filepath.Join(thumbDir, video.Hash+".gif")); err == nil {
			GlobalLogger.WithFields(logrus.Fields{"path": video.Path, "hash": video.Hash}).Info("thumbnail already existing")
			continue
		}

		GlobalLogger.WithFields(logrus.Fields{"path": video.Path, "hash": video.Hash}).Info("thumbnail generating start")
		out, err := makeThumbnail(workingDir, thumbDir, video)
		if err != nil {
			GlobalLogger.WithFields(logrus.Fields{
				"error":  err,
				"output": string(out),
			}).Error("thumnail generating error")
		}
	}
}

func getThumbnailDirectory() (string, error) {
	if ThumbnailDir == "" {
		wdpath, err := os.Getwd()
		if err != nil {
			return "", err
		}
		thumbDir := filepath.Join(wdpath, "thumb")
		if err := os.MkdirAll(thumbDir, os.ModePerm); err != nil {
			return "", err
		}
		return thumbDir, nil
	} else {
		return ThumbnailDir, nil
	}
}

func makeThumbnail(workingPath string, resultPath string, video Video) ([]byte, error) {
	thumbSetName := video.Hash + "%02d.jpg"
	thumbOutputName := video.Hash + ".gif"
	out := []byte{}

	outExtract, errExtract := exec.Command("ffmpeg", "-y", "-ss", "3", "-i", video.Path, "-vf", "'select=gt(scene\\,0.1)'", "-frames:v", "10", "-vsync", "vfr", "-vf", "fps=fps=1/"+strconv.Itoa(ThumbnailMinimumInterval), filepath.Join(workingPath, thumbSetName)).CombinedOutput()
	out = append(out, outExtract...)
	if errExtract != nil {
		return out, errExtract
	}

	outConvert, errConvert := exec.Command("ffmpeg", "-y", "-f", "image2", "-i", filepath.Join(workingPath, thumbSetName), "-framerate", "1", "-vf", "scale=480:-1:flags=lanczos,setpts=8*PTS", filepath.Join(resultPath, thumbOutputName)).CombinedOutput()
	out = append(out, outConvert...)
	if errConvert != nil {
		return out, errConvert
	}

	return out, nil
}

func getVideos(rootPath string, nameFunc NameFunction) ([]Video, error) {
	videoPaths, err := getVideoPaths(rootPath)
	if err != nil {
		return nil, err
	}
	videos := []Video{}
	for _, path := range videoPaths {
		hash, err := nameFunc(path)
		if err != nil {
			return nil, err
		}

		finfo, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		videos = append(videos, Video{
			Path:       path,
			Hash:       hash,
			Name:       filepath.Base(path),
			Size:       finfo.Size(),
			ModifiedAt: finfo.ModTime(),
		})
	}
	return videos, nil
}

func getFileBase64(path string) (string, error) {
	b := base64.StdEncoding.EncodeToString([]byte(path))
	hash := sha1.New()
	hash.Write([]byte(b))
	hashOut := hash.Sum(nil)
	return fmt.Sprintf("%x", hashOut), nil
}

func getFileCRC32(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hashPoly := crc32.MakeTable(crc32.IEEE)
	hashGen := crc32.New(hashPoly)
	if _, err := io.Copy(hashGen, file); err != nil {
		return "", err
	}
	hashBytes := hashGen.Sum(nil)[:]
	hash := hex.EncodeToString(hashBytes)
	return hash, nil
}

func getVideoPaths(rootPath string) ([]string, error) {
	res := []string{}
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.Contains(AllowedExtension, filepath.Ext(info.Name())) {
			res = append(res, path)
		}
		return nil
	})
	return res, err
}

func checkFFmpeg() error {
	return exec.Command("ffmpeg", "-version").Run()
}
