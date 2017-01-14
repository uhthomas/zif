import React, { Component } from 'react';

import Dialog from 'material-ui/Dialog';
import RaisedButton from 'material-ui/RaisedButton';
import Subheader from 'material-ui/Subheader';
import {List, ListItem} from 'material-ui/List';

import ReactList from 'react-list';
import {Wave} from "better-react-spinkit";

import File from "./File";
import util from "../util"

var wjs = require("wcjs-player");


class Playback extends Component
{
	constructor(props) 
	{
		super(props);

		this.state = {
			open: true
		};

		this.componentDidMount = this.componentDidMount.bind(this);
		this.componentWillUnmount = this.componentWillUnmount.bind(this);
	}

	componentDidMount(){
		this.player = new wjs("#player").addPlayer({
			  autoplay: true,
			   wcjs: require('webchimera.js')
			});

			this.player.addPlaylist(this.props.url);

			if (this.playerDOM.style.height != this.playerDOM.parentNode.style.maxHeight){
				// ugghhh. the hacks :/
				this.playerDOM.style.height = this.playerDOM.parentNode.style.maxHeight;
				this.playerDOM.parentNode.style.padding = "0";
				this.playerDOM.parentNode.parentNode.parentNode.style.maxWidth= "";
				this.forceUpdate();
			}
	}

	componentWillUnmount(){
		this.player.stop();
	}

	render() {
		return (<Dialog
		  modal={false}
		  open={this.state.open}
		  onRequestClose={() => {this.setState({ open: false}); this.props.onClose();}}
		  style={{padding: "0", maxWidth: ""}}>
		
		  <div ref={(i) => this.playerDOM = i } id="player" 
		  		style={{width: "100%"}}></div>
		
		</Dialog>)
	}
}

export default Playback;
