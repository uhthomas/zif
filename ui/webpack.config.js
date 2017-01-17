var nodeExternals = require('webpack-node-externals');

module.exports = {
	target: "electron",
	externals: [nodeExternals()],
	entry: './main.js',
		output: {
		filename: 'dist/bundle.js'       
	},

	module: {
		loaders: [
			{ test: /\.coffee$/, loader: 'coffee-loader' },
			{
				test: /\.js$/,
				loader: 'babel-loader',
				query: {
					presets: ['es2015', 'react']
				}
			}
		]
	}
};
