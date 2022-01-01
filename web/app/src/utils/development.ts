//per webapp/config/env.js and webapp/config/webpack.config.dev.js
const isDevelopment = (): boolean => {
  return process.env.NODE_ENV === 'development';
}

const isProduction = (): boolean => {
  return process.env.NODE_ENV === 'production';
}

export const DEVELOPMENT = isDevelopment();
export const PRODUCTION = isProduction();
