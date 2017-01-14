import React, { Component } from 'react';
import { Router, Route, hashHistory, Link } from 'react-router';
import request from "superagent"

import {Toolbar, ToolbarGroup, ToolbarSeparator, ToolbarTitle} from 'material-ui/Toolbar';
import {grey100, grey50} from 'material-ui/styles/colors';
import {RotatingPlane} from "better-react-spinkit";


import Home from './Home';
import Downloads from './Downloads';
import SearchResults from "./SearchResults"
import Stream from "./Stream"
import Welcome from "./WelcomeDialog"
import Playback from "./Playback"

import NavBar from "./NavBar"

import util from "../util"
import hadouken from "../hadouken"

var routes = [{ path: "/", component: Home },
			  { path: "/search", component: SearchResults },
			  { path: "/stream/:infohash", component: Stream },
			  { path: "/downloads", component: Downloads }];

class App extends Component
{

	constructor(props)
	{
		super(props);

		this.state = { drawerOpen : true, 
			search: {
				focus: false
			},
			showSpinner: false
		};

		this.handleToggle = this.handleToggle.bind(this);
		this.onResults = this.onResults.bind(this);
		this.setSpinner = this.setSpinner.bind(this);

		window.config = util.loadConfig();
		window.hadouken = hadouken;
		window.routerHistory = hashHistory;
		window.setSpinner = this.setSpinner;

		window.zifColor = {
			primary: "#3f3b3b",
			secondary: "#eee9d9",
			highlight: "#7f5ab6",
			accent: "#b11106"
		};

		window.entry = { address: {} };

		  
		request.get("http://127.0.0.1:8080/self/get/entry/")
				.end(((err, res) => {
					if (err || res.body.status != "ok")
						return;

					var encoded = window.entry.address.encoded;

					window.entry = JSON.parse(res.body.value);

					if (window.navbar)
						window.navbar.setState({ name: window.entry.name });

					if (encoded) {
						window.entry.address.encoded = encoded;
					}
				}));

		request.get("http://127.0.0.1:8080/self/get/zif/")
				.end(((err, res) => {
					if (err || res.body.status != "ok")
						return;

					window.entry.address.encoded = res.body.value;
				}));
	}

	handleToggle(){ this.setState({ drawerOpen: !this.state.drawerOpen }) }

	setSpinner(on){
		this.setState({ showSpinner: on});
	}

	onResults(res) 
	{
		routes[1].component = () => {
			return (
				<SearchResults posts={res}/>
			)
		};

		hashHistory.push("/search")
	}

	homeButtonClick(){ hashHistory.push("/") }

	render() 
	{
		var style = {
			drawerItems: {
				marginTop: "75px"
			},

			router: {
				marginTop: "10px"
			},
			topbar: {
				backgroundColor: window.zifColor.primary
			}

		}


		return(
			<div style={{height: "100%"}}>
				<Welcome />
				{ this.state.showSpinner == true &&
					<RotatingPlane size={36} style={{position: "fixed", top: "50%", left:"50%"}} />
				}
				<div style={{height: "100%"}}>
					<Router history={hashHistory} routes={routes}>
					</Router>
				</div>
			</div>
		)
	}
}

export default App;
