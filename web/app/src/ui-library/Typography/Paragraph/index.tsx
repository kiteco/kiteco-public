import React from 'react';

import { Typography } from 'antd';
import { ParagraphProps } from 'antd/lib/typography/Paragraph';

const { Paragraph } = Typography;

class CustomParagraph extends React.PureComponent<ParagraphProps> {
  render() {
    return (
      <Paragraph {...this.props as ParagraphProps} />
    );
  }
}

export { CustomParagraph as Paragraph };
