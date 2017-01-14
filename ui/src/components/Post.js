const {ipcRenderer} = require('electron');

import request from "superagent"
import React, { Component } from 'react';
import Chip from 'material-ui/Chip';
import { hashHistory, Link } from 'react-router';
import {Card, CardActions, CardHeader, CardText} from 'material-ui/Card';
import RaisedButton from 'material-ui/RaisedButton';
import {List, ListItem} from 'material-ui/List';

import moment from "moment"
import Stream from "./Stream"
import resolve from "../EntryStore.js"
import PostFocus from "./PostFocus"

const style = {
	marginTop: "10px"
};

import util from "../util"

class Post extends Component
{
	constructor(props){
		super(props);

		this.state = {
			source: props.source,
			showStream: false,
			showFocus: false
		};

		this.onContextMenu= this.onContextMenu.bind(this);

		if (this.props.meta.length > 0)
			this.state.meta = JSON.parse(this.props.meta);
		else
			this.state.meta = {};

	}


	static get defaultProps() {
		return {
			title: "Untitled",
			infohash: "nohash",
			seeders: 0,
			leechers: 0,
			uploadDate: 0,
			fileCount: 0,
			size: 0,
			tags: [],
			source: "foo",
			meta: "{}"
		}
	}

	formatUnixTime(time) {
		var m = moment.unix(time);

		return m.fromNow();
	}

	onContextMenu(e) {
		e.preventDefault();

		var menu = new Menu();	
		menu.append(new MenuItem(
			{ 
				label: "Copy Magnet Link",
				click: () => {
					clipboard.writeText(util.make_magnet(this.props.infohash));
				}
			}));

		menu.append(new MenuItem(
			{ 
				label: "Download",
				click: () => {
					hadouken.addLink(util.make_magnet(this.props.infohash), ()=>{});
				}
			}
		));

		menu.popup(remote.getCurrentWindow());
	}

	componentDidMount(){
		resolve(this.state.source, (err, res) => {
			this.setState({ source: res.name });
		});
	}

	render() {
		return (
			<div className="card" onContextMenu={this.onContextMenu}
					onClick={() => {this.setState({ showFocus: true })}}>
						<div style={{display: "inline"}}>
							<h2 className="title">{this.props.title}</h2>
							<div className="info">
								<div><span style={{color:"#279c10"}}>{this.props.seeders}</span> / <span style={{color:"#b11106"}}>{this.props.leechers}</span></div>
								<div>{this.props.fileCount} files, {util.bytes_to_size(this.props.size)}</div>
								<div>{this.formatUnixTime(this.props.uploadDate)}</div>
							</div>
						</div>
						<div className="source"><em>uploaded by {this.state.source}</em></div>

					<PostFocus title={this.props.title}
								meta={this.props.meta}
								open={this.state.showFocus}
							magnet={util.make_magnet(this.props.infohash)}
							onClose={()=>this.setState({ showFocus: false })}
							source={this.state.source}/>
			</div>)
	}
}

export default Post;
