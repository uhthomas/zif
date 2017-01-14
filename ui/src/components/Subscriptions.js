"use babel";

import React, { Component } from 'react';
import ReactCSSTransitionGroup from 'react-addons-css-transition-group';
import request from "superagent"

import {List, ListItem} from 'material-ui/List';
import FlatButton from 'material-ui/FlatButton';

import Post from "./Post"
import NavBar from "./NavBar"
import Search from "./Search"
import resolve from "../EntryStore.js"

class Subscriptions extends Component{

	constructor(props){
		super(props);

		this.state = { subscriptions: [] };

		this.renderSubscriptions = this.renderSubscriptions.bind(this);
		this.componentDidMount = this.componentDidMount.bind(this);
	}

	static get defaultProps()
	{ 
		return{
			subscriptions: [],
			value: ""
		} 
	}

	componentDidMount(){
		window.config.subscriptions.map((sub, index) => {
			resolve(sub, (err, res) => {
				if (err) return;
				this.setState({ subscriptions: this.state.subscriptions.concat(res) });
				console.log(res);
			});
		});
	}

	renderSubscriptions(){
		var subs = [];
		for (var i = 0; i < this.state.subscriptions.length; i++){
			subs.push(<div className="card">
				<h2 className="title">{this.state.subscriptions[i].name}</h2>
			</div>);

			console.log(i)
		}

		console.log(subs)
			console.log(this.state)

		return subs;
	}

	render() {
		return(
			<div className="content">
				<NavBar />
					<ReactCSSTransitionGroup
						transitionName="postAnim"
						transitionEnterTimeout={500}
						transitionLeaveTimeout={500}
						transitionAppear={true}
						transitionAppearTimeout={500}>

						{this.renderSubscriptions()}
					</ReactCSSTransitionGroup>
			</div>
		)
	}
}

export default Subscriptions;
