import React, { Component } from 'react';
import request from "superagent";

import Dialog from 'material-ui/Dialog';
import FlatButton from 'material-ui/FlatButton';
import Subheader from 'material-ui/Subheader';
import Checkbox from 'material-ui/Checkbox';
import {List, ListItem} from 'material-ui/List';
import TextField from 'material-ui/TextField';

import util from "../util"

class Welcome extends Component
{
	constructor(props)
	{
		super(props);

		this.state = {
			open: !window.config.welcomed,
			bootstrap: {
				zif: false
			}
		};

		this.handleClose = this.handleClose.bind(this);

		this.checkZif = this.checkZif.bind(this);
		this.handleNameChange = this.handleNameChange.bind(this);
		this.handleDescChange = this.handleDescChange.bind(this);

		this.setChannelDetails = this.setChannelDetails.bind(this);
	}

	handleClose() { 
		// Make sure at least one has been selected.
		if (!this.state.bootstrap.zif) return;

		// BOOTSTRAP! :D
		if (this.state.bootstrap.zif) {
			this.bootstrapUrl("vqthcutzukicppty.onion")
		}

		this.setChannelDetails();

		this.setState({ open: !this.state.open });

		window.config.welcomed = true;
		util.saveConfig(window.config);
	}

	bootstrapUrl(url) {
		request.get('http://127.0.0.1:8080/self/bootstrap/' + url + "/")
				.end((error, response) => { console.log(response); })
	}

	setChannelDetails(){
		request.post("http://127.0.0.1:8080/self/set/name/")
				.type("form")
				.send({ value: this.name })
				.end((err, res) => {
				});

		request.post("http://127.0.0.1:8080/self/set/desc/")
				.type("form")
				.send({ value: this.desc})
				.end((err, res) => {
				});
	}

	checkZif(e, checked) {
		this.setState({
			bootstrap: { zif: checked }
		});
	}

	handleNameChange(event){
		this.name = event.target.value;
	}

	handleDescChange(event){
		this.desc = event.target.value;
	}

	render() 
	{
		return (<Dialog
		  title="Welcome to Zif"
		  modal={false}
		  open={this.state.open}
		  onRequestClose={this.handleClose}
		  actions={<FlatButton label="Go"
        						primary={true}
        						onTouchTap={this.handleClose}
        						disabled={!(this.state.bootstrap.zif)}
        	autoScrollBodyContent={true}/>}>

			<p>Enter the details of your channel</p>
			<TextField
				onChange={this.handleNameChange}
				floatingLabelText="Name"
				fullWidth={true}/><br/>

			<TextField
				onChange={this.handleDescChange}
				floatingLabelText="Description"
				multiLine={true}
				fullWidth={true}
				/><br/>


			<p>Pick one or more of the channels below</p>
			<List>
				  <ListItem 
				  		primaryText="zif.io" 
				  		leftCheckbox={<Checkbox onCheck={this.checkZif} />}
				  		secondaryText={
				  			<p>The official Zif channel</p>
				  		}/>
    		</List>,

		</Dialog>)
	}
}

export default Welcome;
