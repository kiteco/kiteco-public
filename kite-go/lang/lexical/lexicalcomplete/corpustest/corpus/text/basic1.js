const mapStateToProps = (state, ownProps) => ({
  system: state.system,
  ...ownProps,
})

const mapDispatchToProps = dispatch => ({
})

// TEST
// export default connect(^
// @0 mapStateToProps
// @1 `mapStateToProps, mapDispatchToProps`
// status: ok
