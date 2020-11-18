package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	elogrus "github.com/dictor/echologrus"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

var (
	GlobalLogger     *logrus.Logger
	AllowedExtension string = ".mkv .mp4 .webm .avi"
)

type (
	Video struct {
		Path string
		Hash string
		Name string
	}

	NameFunction func(string) (string, error)
)

func main() {
	e := echo.New()
	GlobalLogger = elogrus.Attach(e).Logger

	if err := checkFFmpeg(); err != nil {
		GlobalLogger.WithError(err).Fatal("FFmpeg check error")
	}

	videos, err := getVideos("e:/영화/", getFileBase64)
	if err != nil {
		GlobalLogger.WithError(err).Fatal("Video scan error")
	}
	startThumbnailTask(videos)

	e.Logger.Fatal(e.Start(":80"))
}

func startThumbnailTask(videos []Video) {
	workingDir, err := ioutil.TempDir("", "thumbnailer")
	if err != nil {
		GlobalLogger.Fatal(err)
	}
	defer os.RemoveAll(workingDir)

	wdpath, err := os.Getwd()
	if err != nil {
		GlobalLogger.Fatal(err)
	}

	thumbDir := filepath.Join(wdpath, "thumb")
	if err := os.MkdirAll(thumbDir, os.ModePerm); err != nil {
		GlobalLogger.Fatal(err)
	}
	GlobalLogger.WithFields(logrus.Fields{"thumbnail_dir": thumbDir, "temp_dir": workingDir}).Info("thumnail task start")

	for _, video := range videos {
		GlobalLogger.WithFields(logrus.Fields{"path": video.Path, "hash": video.Hash}).Info("thumnail generating start")
		out, err := makeThumbnail(workingDir, thumbDir, video)
		if err != nil {
			GlobalLogger.WithFields(logrus.Fields{
				"error":  err,
				"output": string(out),
			}).Error("thumnail generating error")
		}
	}
}

func makeThumbnail(workingPath string, resultPath string, video Video) ([]byte, error) {
	thumbSetName := video.Hash + "%02d.jpg"
	thumbOutputName := video.Hash + ".gif"
	out := []byte{}

	outExtract, errExtract := exec.Command("ffmpeg", "-y", "-ss", "3", "-i", video.Path, "-vf", "'select=gt(scene\\,0.1)'", "-frames:v", "10", "-vsync", "vfr", "-vf", "fps=fps=1/200", filepath.Join(workingPath, thumbSetName)).CombinedOutput()
	out = append(out, outExtract...)
	if errExtract != nil {
		return out, errExtract
	}

	outConvert, errConvert := exec.Command("ffmpeg", "-y", "-f", "image2", "-i", filepath.Join(workingPath, thumbSetName), "-framerate", "1", "-vf", "scale=480:-1:flags=lanczos,setpts=6*PTS", filepath.Join(resultPath, thumbOutputName)).CombinedOutput()
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
		videos = append(videos, Video{Path: path, Hash: hash, Name: filepath.Base(path)})
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
