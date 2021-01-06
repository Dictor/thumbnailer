var ThumbnailerApp = {
    app: {},
    init: function() {
        ThumbnailerApp.app = new Vue({
            el: '#app',
            data: {
                videos: [],
                pathFilter: "",
                sort: "nameu",
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
                parseCondition: function(c) {
                    const condToProp = {
                        "name": "name",
                        "date": "modified_at",
                        "size": "size",
                    }
                    return function(a, b) {
                        return ((a[condToProp[c]] > b[condToProp[c]]) ? 1 : -1) * ((c[4] === "u") ? 1 : -1);
                    }
                },
            },
            mounted: function() {
                this.getVideos();
            },
            computed: {
                filteredVideo: function() {
                    let res = this.videos.filter(v => v.path.toLowerCase().includes(this.pathFilter.toLowerCase()));
                    res.sort(this.parseCondition(this.sort));
                    return res;
                },
            },
            components: {
                'v-lazy-image': VLazyImage.default,
            },
        });
    }
}