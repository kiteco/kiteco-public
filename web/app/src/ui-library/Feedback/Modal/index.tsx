import React from 'react';

import { Modal } from 'antd';
import { ModalProps } from 'antd/lib/modal';

import './index.less';

class CustomModal extends React.PureComponent<ModalProps> {
  render() {
    return (
      <Modal {...this.props as ModalProps}>
        {this.props.children}
      </Modal>
    );
  }
}

export { CustomModal as Modal };
