import React from 'react'

import demo from './demo.mp4'
import './demo-video.css'

const DemoVideo = () =>
  <div className="demo-video">
    <video autoPlay loop muted playsInline>
      <source src={demo} type="video/mp4"/>
    </video>
  </div>

export default DemoVideo
