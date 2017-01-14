import React, { Component } from 'react';

import Dialog from 'material-ui/Dialog';
import RaisedButton from 'material-ui/RaisedButton';
import Subheader from 'material-ui/Subheader';
import {List, ListItem} from 'material-ui/List';


import ReactList from 'react-list';
import {Wave} from "better-react-spinkit";

import File from "./File";
import util from "../util"
import Stream from "./Stream"

var wjs = require("wcjs-player");


class PostFocus extends Component
{
	constructor(props) 
	{
		super(props);

		this.state = {
			open: true,
			streamOpen: false
		};

		if (this.props.meta.length > 0)
			this.state.meta = JSON.parse(this.props.meta);
		else
			this.state.meta = {};

		this.componentDidMount = this.componentDidMount.bind(this);
	}

	static get defaultProps(){
		return {
			title: "notitle",
			infohash: "nohash",
			meta: "{}",
			onClose: ()=>{}
		}
	}

	componentDidMount(){
	}

	render() {
		return (<Dialog
				  modal={false}
				  open={this.props.open}
				  onRequestClose={() => {this.setState({ open: false}); this.props.onClose();}}
				  style={{padding: "0", maxWidth: ""}}
				  title={[<h2 style={{marginBottom:0, marginLeft: "10px"}}>{this.props.title}</h2>, 
		  	  				<em style={{marginLeft:"35px"}}>uploaded by {this.props.source}</em>]}>

				<div className="body">
					<div className="description">
						<span>{this.state.meta.description != undefined &&
						this.state.meta.description}</span>
					</div>


					<div style={{ float: "right" }}>
						<a style={{display: "block"}}
							onContextMenu={this.onContextMenu}
							href={util.make_magnet(this.props.infohash)}>
							<span>Magnet</span>
						</a>

						<RaisedButton style={{display:"block"}}
							onClick={() => this.setState({streamOpen: true})}>
							Stream
						</RaisedButton>
					</div>

					<Stream open={this.state.streamOpen} magnet={this.props.magnet} onClose={() => this.setState({streamOpen: false})} />

				</div>
		
		</Dialog>)
	}
}

export default PostFocus;
