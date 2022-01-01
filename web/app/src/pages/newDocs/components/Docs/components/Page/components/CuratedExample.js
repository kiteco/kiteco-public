import React from "react";
import ReactMarkdown from "react-markdown";
import { Link } from "react-router-dom";

import { LinedBlock } from "../../util/Code";
import Output from "../../util/Output";
import FileViewer from "./FileViewer";

const CuratedExample = ({
  example,
  id,
  full_name,
  language,
  linkTitle = true
}) => {
  const computeChunk = (chunk, index) => {
    switch (chunk.type) {
      case "code":
        return (
          <LinedBlock
            key={`${id}-${index}`}
            numberLines={false}
            code={chunk.content.code}
            references={chunk.content.references}
            highlightedIdentifier={full_name}
            language={language}
          />
        );
      case "output":
        return <Output key={`${id}-${index}`} chunk={chunk} />;
      default:
        return null;
    }
  };
  return (
    <section className="how-to">
      <h4>
        {linkTitle && (
          <Link
            to={`/${language}/examples/${id}/${example.package}-${example.title
              .toLowerCase()
              .replace(/\s/g, "-")
              .replace("%", "percent-sign")}`}
          >
            <ReactMarkdown source={example.title} />
          </Link>
        )}
      </h4>
      <pre className="with-background">
        <code className="with-syntax-highlighting code">
          <div className="example-prelude">
            {example.prelude.map(computeChunk)}
          </div>
          <div className="example-main">{example.code.map(computeChunk)}</div>
          <div className="example-postlude">
            {example.postlude.map(computeChunk)}
          </div>
          {example.inputFiles && example.inputFiles.length > 0 && (
            <div className="example-input-files">
              <h3 className="example-input-files__header">Input Files</h3>
              {example.inputFiles.map((file, index) => (
                <FileViewer
                  key={`${file.name}-${index}`}
                  data={file}
                  language={language}
                />
              ))}
            </div>
          )}
        </code>
      </pre>
    </section>
  );
};

export default CuratedExample;
