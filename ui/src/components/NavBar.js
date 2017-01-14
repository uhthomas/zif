import React, { Component } from 'react';
import { Router, Route, hashHistory, Link } from 'react-router';
import request from "superagent"

import {List, ListItem} from 'material-ui/List';
import Divider from 'material-ui/Divider';
import Subheader from 'material-ui/Subheader';
import FontIcon from 'material-ui/FontIcon';
import IconButton from 'material-ui/IconButton';

import util from "../util.js"
import Upload from "./UploadDialog.js"
import EditAccount from "./EditAccount.js"
import ToolTip from 'react-portal-tooltip'

class NavBar extends Component{

	constructor(props){
		super(props);


		this.state = { name: window.entry.name, uploadTooltip: false, channelTooltip: false, homeTooltip: false };
		window.navbar = this;
		this.uploadDialog = <Upload />;

		this.upload = {};
	}

	toggleUploadTooltip(){
		this.setState({ uploadTooltip: !this.state.uploadTooltip, channelTooltip: false, homeTooltip: false })
	}

	toggleHomeTooltip(){
		this.setState({ homeTooltip: !this.state.homeTooltip, channelTooltip: false })
	}

	toggleChannelTooltip(){
		request.get("http://127.0.0.1:8080/self/get/postcount/")
				.end(((err, res) => {
					if (err || res.body.status != "ok")
						return;

					window.entry.postCount = res.body.value;
					this.setState({});
				}).bind(this));

		this.setState({ channelTooltip: !this.state.channelTooltip, uploadTooltip: false, homeTooltip: false })
	}

	static get defaultProps(){
		return {
		
		}
	}

	render(){
		return(
				<ul className="topnav" id="mainMenu">
					<li><a 
							href="#" 
							id="logo"
							onMouseEnter={() => this.setState({homeTooltip: true})}
							onMouseLeave={() => this.setState({homeTooltip: false})}
							onClick={() => this.setState({homeTooltip: false})}>Zif</a></li>

					<li><a href="#downloads">Downloads</a></li>

					<li style={{float: "right"}}>
						<a id="channel"
							onMouseEnter={this.toggleChannelTooltip.bind(this)}
							onMouseLeave={this.toggleChannelTooltip.bind(this)}>
							{this.state.name}
						</a>
					</li>

					<li style={{float: "right"}}>
						<a id="upload" 
								onMouseEnter={this.toggleUploadTooltip.bind(this)}
								onMouseLeave={this.toggleUploadTooltip.bind(this)}
								onClick={this.upload.open}>
							<i className="material-icons">file_upload</i>
						</a>
					</li>

					{this.props.advancedClick && 
					<li style={{float: "right"}}>
						<a id="advancedDownload" 
							onClick={this.props.advancedClick}>
							<i className="material-icons">flash_on</i>
						</a>
					</li>
					}


					<ToolTip active={this.state.uploadTooltip} 
							position="bottom" 
							arrow="center" 
							parent="#upload">
						<div>
							Upload
						</div>
					</ToolTip>

					<ToolTip active={this.state.homeTooltip} 
							position="bottom" 
							arrow="center" 
							parent="#logo">
						<div>
							Home
						</div>
					</ToolTip>

					<ToolTip active={this.state.channelTooltip} 
							position="bottom" 
							arrow="center" 
							parent="#channel">
						<div>
							<List>
								<ListItem
								  primaryText="Address"
								  secondaryText={window.entry.address.encoded}/>
								<Divider />
								<ListItem
								  primaryText="Description"
								  secondaryText={window.entry.desc}
								  secondaryTextLines={2}/>
								<Divider />
								<ListItem
								  primaryText={<div> {window.entry.postCount} Posts</div>}/>
							</List>
						</div>
					</ToolTip>

					<Upload nav={this} />
					<EditAccount nav={this} />
				</ul>

			  )
	}
}

export default NavBar;
