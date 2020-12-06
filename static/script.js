var ThumbnailerApp = {
    app: {},
    init: function() {
        ThumbnailerApp.app = new Vue({
            el: '#app',
            data: {
                videos: [],
                pathFilter: "",
            },
            methods: {
                getVideos: async function() {
                    this.videos.splice(0);
                    for (v of(await axios.get("./video")).data) {
                        this.videos.push(v);
                    }
                },
                getThumbPath: function(hash) {
                    return "./thumb/" + hash + ".gif";
                },
                moveVideo: function(v) {
                    let s = v.path;
                    s = s.replace(/\\/gi, "/");
                    window.open("/prfile/" + s.substring(s.indexOf("/") + 1));
                },
            },
            mounted: function() {
                this.getVideos();
            }
        });
    }
}