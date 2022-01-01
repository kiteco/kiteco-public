require('./global-utils.js');

function findAll(x, f) {
  if (Array.isArray(x)) {
    var r = [];
    for (var i = 0; i < x.length; i++) {
      if (f(x[i])) {
        r.push(x[i]);
      }
    }
    return r;
  } else {
    console.error("Tried passing non-array object to findAll");
  }
}

// from: SO link: /questions/3169786/clear-text-selection-with-javascript
function deselectAll() {
  if (window.getSelection) {
    if (window.getSelection().empty) {  // Chrome
      window.getSelection().empty();
    } else if (window.getSelection().removeAllRanges) {  // Firefox
      window.getSelection().removeAllRanges();
    }
  } else if (document.selection) {  // IE?
    document.selection.empty();
  }
}

// ugh this shouldn't be here...
function convertStatus(s) {
  if (s === 'in_progress') return 'In progress';
  if (s === 'pending_review') return 'Pending review';
  if (s === 'needs_attention') return 'Needs attention';
  if (s === 'approved') return 'Approved';

  if (s === 'In progress') return 'in_progress';
  if (s === 'Pending review') return 'pending_review';
  if (s === 'Needs attention') return 'needs_attention';
  if (s === 'Approved') return 'approved';

  return s;
}

var CheckboxFilter = React.createClass({
  getInitialState: function() {
    var initialSelection = this.props.initialSelection || this.props.options;
    return {
      options: map(this.props.options, (opt) => {
        return { name: opt, show: initialSelection.includes(opt) };
      }),
    };
  },
  changeFilter: function(evt) {
    var newOpts = map(this.state.options, (opt) => {
      if (opt.name === evt.target.name) {
        return { name: opt.name, show: !opt.show };
      } else {
        return opt;
      }
    });
    this.setState({ options: newOpts });
    this.props.onFilterChanged(map(findAll(newOpts, opt => opt.show), opt => opt.name));
  },
  filterOnly: function(evt) {
    evt.preventDefault();
    var optionName = $(evt.target).parent('label').find('input[type="checkbox"]').attr('name');
    var newOpts = map(this.state.options, (opt) => {
      return { name: opt.name, show: opt.name === optionName };
    });
    this.setState({ options: newOpts });
    this.props.onFilterChanged(map(findAll(newOpts, opt => opt.show), opt => opt.name));
    deselectAll();
  },
  render: function() {
    return <div className="checkbox-filter">
      {this.state.options.map(opt =>
        <label onDoubleClick={this.filterOnly}>
          <input type="checkbox"
            checked={opt.show}
            name={opt.name}
            onChange={this.changeFilter} />
          {convertStatus(opt.name)}
        </label>
      )}
    </div>;
  },
});

module.exports = CheckboxFilter;
