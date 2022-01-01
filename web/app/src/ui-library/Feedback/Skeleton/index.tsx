import React from 'react';

import { Skeleton } from 'antd';
import { SkeletonProps } from 'antd/lib/skeleton';

import './index.less';

class CustomSkeleton extends React.PureComponent<SkeletonProps> {
  render(): JSX.Element {
    return (
      <Skeleton {...this.props}>
        {this.props.children}
      </Skeleton>
    );
  }
}

export { CustomSkeleton as Skeleton };
