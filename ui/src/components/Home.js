import React, { Component } from 'react';
import request from "superagent"

import FontIcon from 'material-ui/FontIcon';

import NavBar from "./NavBar"
import Search from "./Search"
import ToolTip from 'react-portal-tooltip'

class Home extends Component{

	constructor(props){
		super(props);

		this.state = {
		};

		this.config = this.props.config;

		if(!this.props.Posts) this.state.posts = [];
		else this.state.posts = this.props.Posts;
	}

	static get defaultProps()
	{ 
		return{
			posts: []
		} 
	}

	componentDidMount() {
		this.getPosts = request.get("http://127.0.0.1:8080/self/popular/0/")
						.accept("json")
						.type("json")
						.end((err, res) => {
							if (err) {
								return console.log(err);
							}
							this.setState({posts: res.body.value});
						});

	}

	componentWillUnmount() {
		this.getPosts.abort()
	}

	render() {
		return(

			<div >
				<NavBar />
				<div className="outer">
					<div className="middle">
						<div className="inner">
							<Search
								onResults={
									(res, query) => {
										this.props.router.push({
											pathname: "/search",
											state: {
												posts: res,
												query: query
											}
										});
									}
								}
							/>
						</div>
					</div>
				</div>
			</div>
		)
	}
}

export default Home;
