import React from 'react';

import { Layout } from '../../../ui-library/Layout';
import { Header } from '../../../ui-library/Layout/Header';
import { Content } from '../../../ui-library/Layout/Content';
import { Footer } from '../../../ui-library/Layout/Footer';

import styles from './index.module.less';

import { ReactComponent as KiteLogo } from './images/kite-dark-logo-with-text.svg';
import heart from './images/heart.png';

export default class BasicLayout extends React.PureComponent {
  render() {
    return (
      <Layout>
        <Header>
          <div className="container">
            <a href="/" className={styles.logo} tabIndex={-1}>
              <KiteLogo />
            </a>
          </div>
        </Header>
        <Content>
          {this.props.children}
        </Content>
        <Footer>
          <div className="container">
            <p className={styles['love-text']}>
              Made with <img src={heart} alt="Love" width="20" height="20" /> in San Francisco
            </p>
          </div>
        </Footer>
      </Layout>
    );
  }
}
