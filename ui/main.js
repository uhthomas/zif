'use strict';

var electron = require('electron');
var {app, BrowserWindow, ipcMain} = electron;
var spawn = require("child_process").spawn;
var fs = require("fs")

var torrentStream = require("./src/torrent/stream.js");
var zifLog = fs.createWriteStream('./zifd.log', { flags: 'a' })

let mainWindow;
let torrent;
let hadouken;
let zifd;

if(process.platform == "win32"){
    process.env["VLC_PLUGIN_PATH"] = path.join(__dirname, "node_modules/webchimera.js/bin/plugins");
}


function runHadouken() {
	hadouken = spawn("./hadouken", { cwd: "./hadouken" });

	hadouken.stdout.on("data", (data) => {
		console.log("[hadouken]", data.toString());
	});

	hadouken.stderr.on("data", (data) => {
		console.log("[hadouken]", data.toString());
	});

	torrent = torrentStream(ipcMain);
}

function runZifd() {
	zifd = spawn("./zifd");

	zifd.stdout.on("data", (data) => {
		console.log("[zifd]", data.toString());
		zifLog.write(data +"\n", ()=>false);
	});

	zifd.stderr.on("data", (data) => {
		console.log("[zifd]", data.toString());
		zifLog.write(data + "\n", ()=>false);
	});
}

function createWindow () {
	mainWindow = new BrowserWindow({width: 800, height: 600});

	mainWindow.loadURL('file://' + __dirname + '/dist/index.html');
	//mainWindow.setMenu(null);

	mainWindow.on('closed', function() {
		mainWindow = null;
	});

	console.log("Starting zifd...")
	runZifd();
	console.log("Starting hadouken...")
	runHadouken();
}

app.on('ready', createWindow);

app.on('window-all-closed', function () {
	if (process.platform !== 'darwin') {
		app.quit();
	}
});

app.on('activate', function () {
	if (mainWindow === null) {
		createWindow();
	}
});

