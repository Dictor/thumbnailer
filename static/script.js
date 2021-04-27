var ThumbnailerApp = {
    app: {},
    init: function() {
        ThumbnailerApp.app = new Vue({
            el: '#app',
            data: {
                videos: [],
                pathFilter: "",
                sort: "nameu",
                viewType: "video"
            },
            methods: {
                getVideos: async function() {
                    this.videos.splice(0);
                    for (v of(await axios.get("./video")).data) {
                        v.path  = v.path.replace(/\\/gi, "/");
                        this.videos.push(v);
                    }
                },
                getThumbPath: function(hash) {
                    return "./thumb/" + hash + ".gif";
                },
                moveVideo: function(v) {
		            let s = v.path;
                    s = s.replace(/\\/gi, "/");
                    s = s.replace("/video/", "");
                    window.open("/prfile/" + s);
                },
                moveFolder: function(f) {
                    this.pathFilter = f;
                    this.viewType = "video";
                },
                parseCondition: function(c) {
                    return function(a, b) {
			let res = 0;
			switch(c.substring(0, 4)) {
			    case "size":
				res = a.size - b.size;
				break;
			    case "date":
				res = (new Date(a.modified_at)).getTime() - (new Date(b.modified_at)).getTime();
			    	break;
			    case "name":
				res = ((a.name > b.name) ? 1 : -1);
				break;
			}
                        return (res * ((c[4] === "u") ? 1 : -1));
                    }
                },
            },
            mounted: function() {
                this.getVideos();
            },
            filters: {
                folderName: function(folderPath) {
                    const ppath = folderPath.replace(/\\/gi, "/").split("/");
                    return ppath[ppath.length - 1];
                }
            },
            computed: {
                filteredVideo: function() {
                    const res = this.videos.filter(v => v.path.toLowerCase().includes(this.pathFilter.toLowerCase()));
                    res.sort(this.parseCondition(this.sort));
                    return res;
                },
                folderedVideo: function() {
                    const res = {};
                    for (const v of this.videos) {
                        const ppath = v.path.replace(/\\/gi, "/").split("/");
                        const folder = ppath.slice(0, ppath.length - 1).join('/');
                        if (folder in res) {
                            res[folder].push(v);
                        } else {
                            res[folder] = [v];
                        }
                    }
                    console.log(res)
                    return res;
                }
            },
            components: {
                'v-lazy-image': VLazyImage.default,
            },
        });
    }
}
