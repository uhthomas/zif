"use babel";

import React, { Component } from 'react';
import ReactCSSTransitionGroup from 'react-addons-css-transition-group';
import request from "superagent"

import {List, ListItem} from 'material-ui/List';
import FlatButton from 'material-ui/FlatButton';

import Post from "./Post"
import NavBar from "./NavBar"
import Search from "./Search"

class SearchResults extends Component{

	constructor(props){
		super(props);


		this.state = {};
		this.state.posts = this.sortPosts(this.props.location.state.posts);
		console.log(this.state.posts);
	}

	static get defaultProps()
	{ 
		return{
			posts: [],
			value: ""
		} 
	}

	sortPosts(posts) {
		var newPosts = [];

		for(var i = 0; i < posts.length; i++)
		{
			if (!posts[i]) continue;

			for (var j = 0; j < posts[i].posts.length; j++) {
				posts[i].posts[j].source = posts[i].source;
				newPosts.push(posts[i].posts[j]);
			}
		}

		newPosts.sort((a, b) => {
			var aScore = (a.Seeders * 1.1) + a.Leechers;
			var bScore = (b.Seeders * 1.1) + b.Leechers;

			return bScore - aScore;
		});

		return newPosts;
	}

	render() {
		const results = this.state.posts.map((post, index) => {

						return (
							<Post key={post.InfoHash}
								  title={post.Title}
								  infohash={post.InfoHash}
								  seeders={post.Seeders}
								  leechers={post.Leechers}
								  meta={post.Meta}
								  size={post.Size}
								  fileCount={post.FileCount}
								  tags={post.Tags}
								  uploadDate={post.UploadDate}
								  source={post.source}/>
						)
					});
		return(

			<div className="content">
				<NavBar />
				<Search 
					query={this.props.location.state.query} 
					router={this.props.router}
					onResults={(res, query) => {
						this.setState({
							posts: this.sortPosts(res)
						})
					}}/>

					<ReactCSSTransitionGroup
						transitionName="postAnim"
						transitionEnterTimeout={500}
						transitionLeaveTimeout={500}
						transitionAppear={true}
						transitionAppearTimeout={500}>

						{results}
					</ReactCSSTransitionGroup>
			</div>
		)
	}
}

export default SearchResults;
