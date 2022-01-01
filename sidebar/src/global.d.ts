import Electron from 'electron';

declare module '*.module.css' {
  const classes: { [key: string]: string };
  export default classes;
}

declare global {
  interface Window {
    require(moduleSpecifier: 'electron'): typeof Electron;
  }
}
