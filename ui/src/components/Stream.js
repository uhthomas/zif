import React, { Component } from 'react';

import Dialog from 'material-ui/Dialog';
import RaisedButton from 'material-ui/RaisedButton';
import Subheader from 'material-ui/Subheader';
import {List, ListItem} from 'material-ui/List';
import {FoldingCube} from "better-react-spinkit";

import ReactList from 'react-list';

import Playback from "./Playback";
import util from "../util"

import {ipcRenderer} from "electron";


class Stream extends Component
{
	constructor(props) 
	{
		super(props);
		
		this.state = { open: this.props.open, files: [], playback:  false, torrent: null };

		this.componentDidMount = this.componentDidMount.bind(this);
		this.componentWillUnmount = this.componentWillUnmount.bind(this);
		this.sortFiles = this.sortFiles.bind(this);
		this.renderItem = this.renderItem.bind(this);
		this.onTorrent = this.onTorrent.bind(this);
		this.findIndex = this.findIndex.bind(this);

		this.playback = (<Playback open={ this.state.playback } />);
	}

	static get defaultProps(){
		return {
			title: "notitle",
			open: false,
			magnet: "nomagnet"
		}
	}

	sortFiles(files) {
		return files.sort(
				(a, b) => {
					return util.sort.alphanum(a.path, b.path);
				});
				
	}

	// find the index of a file in the original, non-sorted array (needed for
	// the stream url
	findIndex(path) {
		for(var i = 0; i < this.state.torrent.files.length; i++) {
			if (this.state.torrent.files[i].path == path)
				return i;
		}
	}

	onTorrent(e, arg){
		this.setState({ files: arg.torrent.files, torrent: arg.torrent, port: arg.port });
		console.log(arg)
	}

	componentDidMount(){
		this.state.open = this.props.open;

		ipcRenderer.on("torrent", this.onTorrent);

		if (!this.torrent)
			ipcRenderer.send("stream-magnet", this.props.magnet);
	}

	componentWillUnmount() {
		ipcRenderer.removeListener("torrent", this.onTorrent);
	}

	renderItem(index, key) {
		return <ListItem key={key}
						 onClick={()=> { this.setState({
						 	 playback: true,
						 	 streamFile: this.state.files[index],
						 	 index: this.findIndex(this.state.files[index].path)
						 }) }}>{
			this.state.files[index].path
		}</ListItem>;
	}

	render() {
		return (

			<Dialog open={this.props.open} onRequestClose={() => this.props.onClose()}
					autoScrollBodyContent={true}>

					{this.state.files.length == 0 &&
						<div style={{textAlign: "center"}}>
							<h3>Loading files...</h3>
							<FoldingCube style={{display: "inline-block"}} />
						</div>
					}

				<ReactList 
					itemRenderer={this.renderItem}
					length={this.state.files.length}
					type='uniform'/>

				{ this.state.playback && 
					<Playback file={this.state.streamFile} url={"http://localhost:" + this.state.port + "/" + this.state.index}
								onClose={()=>{this.setState({ playback: false})}}/>
				}
			</Dialog>
		)
	}
}

export default Stream;
