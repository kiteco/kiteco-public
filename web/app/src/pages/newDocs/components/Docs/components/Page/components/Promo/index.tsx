import React from 'react'
import { connect } from 'react-redux'

import DemoVideo from './DemoVideo'
import { sendDownloadAnalytics } from '../../../../../../../../components/DownloadButton'
import { TOGGLE_GIF_COLLAPSE } from '../../../../../../../../redux/actions/promotions'

import { navigatorOs } from "../../../../../../../../utils/navigator"
import { getTrackingParamsFromURL } from '../../../../../../../../utils/analytics'

interface IComponentProps {
  collapsedGIF: boolean;
  collapseGIF: any;
}

const os = navigatorOs()

const Promo = ({ collapsedGIF = false, collapseGIF }: IComponentProps) => {
  function getTrackingParams(): string {
    const clickCTA: string = 'promo';

    return getTrackingParamsFromURL('python/docs/', clickCTA)
      || getTrackingParamsFromURL('python/answers/', clickCTA)
      || getTrackingParamsFromURL('python/examples/', clickCTA);
  }

  return (
    <section className='examples-from-your-code promo'>
      <h3>
        <span>
          Want to code faster?
      </span>
        <button onClick={collapseGIF}>
          {collapsedGIF ? "+" : "⌃"}
        </button>
      </h3>
      <div className='text'>
        {!collapsedGIF &&
          <DemoVideo />
        }
      Kite is a plugin for <a href="https://www.kite.com/integrations">any IDE</a> that uses
      deep learning to provide you with intelligent code completions in Python and JavaScript.
      Start coding faster today.
      <div className='actions'>
          <button onClick={() => {
            window.open(`/download${getTrackingParams()}`, "_blank")
            sendDownloadAnalytics()
          }}>{`${os === 'linux' ? "Install" : "Download"} Kite Now! It's Free`}</button>
        </div>
      </div>
    </section>
  );
}

const mapStateToProps = (state: any, ownProps: any) => ({
  collapsedGIF: state.promotions.docsPromoGIFcollapsed,
})

const mapDispatchToProps = (dispatch: any) => ({
  collapseGIF: () => dispatch({ type: TOGGLE_GIF_COLLAPSE }),
})

export default connect(mapStateToProps, mapDispatchToProps)(Promo)
