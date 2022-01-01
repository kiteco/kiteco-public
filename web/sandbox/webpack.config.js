'use strict'

const path = require('path')
const webpack = require('webpack')

const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const TerserJSPlugin = require('terser-webpack-plugin')
const OptimizeCSSAssetsPlugin = require('optimize-css-assets-webpack-plugin')

const config = env => {
  const theme = env && env.kiteTheme
    ? `"${env.kiteTheme}"`
    : '"kite-dark"'
  const configObj = {
    mode: 'production',
    entry: './index.js',
    context: __dirname,
    output: {
      filename: (env && env.kiteTheme) ? `${env.kiteTheme}.js` : 'kite-dark.js',
      path: path.resolve(__dirname, 'dist')
    },
    plugins: [
      // can run `npm run compile -- --env.kiteTheme=kite-light` to set this variable
      new webpack.DefinePlugin({
        'window.kiteTheme': theme
      }),
      new MiniCssExtractPlugin({
        filename: '[name].css',
      }),
    ],
    optimization: {
      minimizer: [new TerserJSPlugin({}), new OptimizeCSSAssetsPlugin({})],
      splitChunks: {
        cacheGroups: {
          styles: {
            name: 'styles',
            test: /\.css$/,
            chunks: 'all',
            enforce: true,
          }
        }
      }
    },
    module: {
      rules: [
        {
          test: /\.js$/,
          exclude: /(node_modules)/,
          use: {
            loader: 'babel-loader',
          }
        },
        {
          test: /\.css$/,
          use: [MiniCssExtractPlugin.loader, 'css-loader', 'postcss-loader']
        },
        {
          test: /\.(png|svg|jpg|gif)$/,
          use: [
            {
              loader: 'file-loader',
              options: {
                name: '[name].[ext]',
                outputPath: 'assets',
                // can run `npm run compile -- --env.assetsPath=something` to set this variable
                publicPath: (env && env.assetsPath) || '/wp-content/kite-sandbox/assets'
              }
            }
          ]
        }
      ]
    }
  }
  return configObj
}

module.exports = config
