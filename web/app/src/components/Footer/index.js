import React from 'react'
import { Link } from 'react-router-dom'

import Newsletter from './Newsletter'
import { Domains } from '../../utils/domains'

import iconLove from './assets/icon-heart.png'
import './assets/footer.css'

const Footer = props =>
  <div className={`footer homepage__section ${props.className ? props.className : ''}`}>
    <div className="homepage__section__content">
      <div className="footer__menu homepage__flex__wrapper">
        <div className="footer__section">
          <h4>Company</h4>
          <a className="footer__menu__item" href={"/"}>Home</a>
          <a className="footer__menu__item" href={"/about"}>About Us</a>
          <a className="footer__menu__item" href={"/careers"}>Careers</a>
          <a className="footer__menu__item" href={"/privacy"}>Privacy</a>
          <a className="footer__menu__item" href={"/contact"}>Contact Us</a>
        </div>
        <div className="footer__section">
          <h4>Product</h4>
          <a
            target="_blank"
            className="footer__menu__item"
            href={"https://www.youtube.com/watch?v=WQUjOgxeSA0"}
            rel="noopener noreferrer"
          >
            Watch a Demo
          </a>
          <a className="footer__menu__item" href={"/integrations"}>Editor Integrations</a>
          <a className="footer__menu__item" href={"/letmeknow"}>Programming Languages</a>
          <Link className="footer__menu__item" to={"/python/docs"}>Python Documentation</Link>
        </div>
        <div className="footer__section">
          <h4>Resources</h4>
          <a className="footer__menu__item" href={"/blog"}>Blog</a>
          <a target="_blank" className="footer__menu__item" href={`http://${Domains.Help}`} rel="noopener noreferrer">Help Center</a>
          <a className="footer__menu__item" href={"/press"}>Press</a>
        </div>
        <div className="footer__section">
          <h4>Stay in touch</h4>
          <div className="footer__social">
            <a target="_blank" className="footer__social-item footer__twitter" href="https://twitter.com/kitehq" alt="twitter" rel="noopener noreferrer"> </a>
            <a target="_blank" className="footer__social-item footer__facebook" href="https://www.facebook.com/Kite-878570708838455/" alt="facebook" rel="noopener noreferrer"> </a>
            <a target="_blank" className="footer__social-item footer__linkedin" href="https://www.linkedin.com/company/kite-co-/" alt="linkedin" rel="noopener noreferrer"> </a>
            <a target="_blank" className="footer__social-item footer__youtube" href="https://www.youtube.com/channel/UCxVRDu9ujwOrmDxu72V3ujQ" alt="linkedin" rel="noopener noreferrer"> </a>
          </div>
          <p className="footer__text">Get Kite updates &amp; coding tips</p>
          <Newsletter/>
        </div>
      </div>
      <div className="footer__disclaimer text__centered">
        Made with <img className="footer__disclaimer__love" src={iconLove} alt="love"/> in San Francisco
      </div>
    </div>
  </div>

export default Footer
