import React from "react";
import { Link } from "react-router-dom";

class Breadcrumbs extends React.Component {
  render() {
    const { items, refFromParent } = this.props;
    return (
      <div ref={refFromParent} className="breadcrumbs">
        {items.map((item, i) => (
          <span
            itemProp="itemListElement"
            itemScope
            itemType="http://schema.org/ListItem"
            key={i}
          >
            <Link to={item.path} itemProp="item">
              <span itemProp="name">{item.name}</span>
              <meta itemProp="position" content={i + 1} />
            </Link>
            <span className="small-text"> &#10093; </span>
          </span>
        ))}
      </div>
    );
  }
}

const BreadcrumbWrapper = ({ children }) => (
  <div itemScope itemType="http://schema.org/BreadcrumbList">
    {children}
  </div>
);

const StarHeader = ({
  moduleName,
  name,
  type,
  isStarred,
  clickHandler,
  path,
  position
}) => (
    <h1
      itemProp="itemListElement"
      itemScope
      itemType="http://schema.org/ListItem"
      className="star-header"
    >
      <Link to={path} itemProp="item">
        <span itemProp="name">
          {moduleName ? moduleName : ""}
          {name ? "." + name : ""}
        </span>
      </Link>
      <div
        className={isStarred ? "star filled" : "star"}
        onClick={clickHandler}
      />
      <meta itemProp="position" content={position} />
    </h1>
  );

const StarHowToHeader = ({ title, isStarred, clickHandler }) => {
  title = stripBackticks(title);
  return (
    <h1 className="star-header">
      How to: {title}
      <div
        className={isStarred ? "star filled" : "star"}
        onClick={clickHandler}
      />
    </h1>
  );
};

const TypeIndicator = ({ type }) => <div className="type">{type}</div>;

const getMetaDescription = (doc, name) => {
  return `Python documentation for ${
    doc.ancestors.length > 0 ? `${doc.ancestors[0].name}.${name}` : name
    }, powered by Kite`;
};

const stripBackticks = (input) => {
  if (input) {
    return input.replace(/`/g, "");
  }
  return input;
}

export {
  Breadcrumbs,
  StarHeader,
  TypeIndicator,
  getMetaDescription,
  stripBackticks,
  BreadcrumbWrapper,
  StarHowToHeader,
};
