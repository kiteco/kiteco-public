import React from 'react';
import ReactMarkdown from 'react-markdown/with-html';

class Description extends React.Component {
    render() {
        const description = this.props.description;
        return <ReactMarkdown source={description} escapeHtml={false}/>
    }
}

export default Description;