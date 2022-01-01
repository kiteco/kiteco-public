const path = require('path');

const config = {
  output: {
    path: path.join(__dirname, '..', 'dist'),
    filename: 'kite-answers-renderer.js',
    library: 'kite-answers-renderer',
    libraryTarget: 'umd',
    publicPath: '/dist/',
    umdNamedDefine: true
  },
  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        loaders: [
          'babel-loader',
        ],
        exclude: /node_modules/,
      },
      {
        test: /\.css$/,
        use: ["style-loader", "css-loader"]
      },
      {
        test: /\.(ico|eot|otf|webp|ttf|woff|woff2)(\?.*)?$/,
        use: [
            {
                loader: 'url-loader',
                options:{
                    fallback: "file-loader",
                    name: "[name][md5:hash].[ext]",
                }
            }    
        ]
      }
    ],
  },
  resolve: {
    alias: {
      'react': path.resolve(__dirname, './node_modules/react'),
      'react-dom': path.resolve(__dirname, './node_modules/react-dom'),
    }
  },
  externals: {
    // Don't bundle react or react-dom
    react: {
        commonjs: "react",
        commonjs2: "react",
        amd: "React",
        root: "React"
    },      
    "react-dom": {
        commonjs: "react-dom",
        commonjs2: "react-dom",
        amd: "ReactDOM",
        root: "ReactDOM"
    }
  }
};

module.exports = config;