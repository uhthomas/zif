var torrent = require("webtorrent");
var http = require("http");
var util = require("../util");

function torrentStream(ipc){
	var home = process.env[(process.platform == 'win32') ? 'USERPROFILE' : 'HOME'];

	var ret = {};
	ret.ipc = ipc;
	ret.client = new torrent();

	ipc.on("stream-magnet", (e, arg) => {
		function add(){
			var t = ret.client.get(arg);

			if (t) {
				ret.server = t.createServer();

				ret.server.listen(0, () => {
					e.sender.send("torrent", { torrent: t, port: ret.server.address().port });
				});

				return;
			}

			ret.client.add(arg, {path: home + "/Downloads"}, (torrent) => {
				console.log("downloading", torrent.infoHash);

				ret.server = torrent.createServer(0);

				console.log("starting to listen")
				ret.server.listen(0, () => {
					console.log("listening")
					e.sender.send("torrent", { torrent: torrent, port: ret.server.address().port });
				});
			});
		}

		if (ret.server)
			ret.server.close(add);
		else
			add()

	});

	return ret;
}

module.exports = torrentStream;
