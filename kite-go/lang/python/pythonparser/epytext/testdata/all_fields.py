# this is to test how fields are grouped by epydoc.
# epydoc considers the section and paragraph that
# follows i1 to be part of i1's description.
#
# - ambiguous variable references (e.g. @param m1 and @var m1), last one wins.
# - type of non-existing var: implicitly declares the parameter with that name.
# - type without a param: dropped probably because not a docstring for a property,
#   otherwise would describe that property. But we don't have that context, so it
#   should be displayed as standalone entry for us.
# - type that appears multiple times for a variable: uses the first.
# - variables without types don't seem to be rendered.
# - group and sort are fields that describe how other symbols in the module/class/etc.
#   get rendered, has no output by themselves (and not useful for us without more context).
# - summary applies only to the short summary next to the python symbol name (top of the
#   rendered HTML document). We can't render this section without more context.
# - known fields are rendered in that order:
#   * inside a single <dl>
#     * Parameters
#     * Returns
#     * Raises
#   * inside the same parent <div>, each in a <p> (or <ul> if appears multiple times):
#     * Note
#     * Version
#     * Todo (followed by Todo(version))
#     * See Also
#     * Requires
#     * Precondition
#     * Postcondition
#     * Invariant
#     * Status
#     * Change Log
#     * Permissions
#     * Bugs
#     * Since
#     * Attention
#     * Deprecated
#     * Author
#     * Organization
#     * Copyright
#     * Warnings
#     * License
#     * Contact
#
def sample():
  """
  This is a description. Fields are supposed to be at the end of the docstring,
  but we can't guarantee that will be the case, so we support fields anywhere.
  Let's see how epydoc behaves when fields are found elsewhere.

  @note: a first note
  @version: v1.0.1
  @ivar i1: class instance variable i1.
  @cvar c1: class static variable c1.
  @type i1: type of variable i1, integer.

  ## A section

  Another paragraph. And finally the bottom of the docstring.

  @rtype: type of return value, double.
  @param a: parameter a description.
  @todo v1.0.1: fix bugs.
  @ivar i2: class instance variable i2.
  @type i2: type of instance variable i2.
  @type c2: type of class variable c2.
  @todo v1.0.2: fix more bugs.
  @todo: a version-less todo.
  @see: something else to see.
  @todo: another version-less todo.
  @type c1: type of variable c1, bool.
  @type c1: 2nd type of variable c1, not a bool.
  @param b: parameter b description.
  @var m1: module variable m1.
  @type m1: type of module variable m1, char.
  @var m2: module variable m2.
  @type m2: type of module variable m2.
  @Type a: type of a, integer, field uppercase
  @requires: file access
  @unknownField: some description for unknown field.
  @param m1: parameter m1, conflicts with module variable m1.
  @summary: long description of what it does, should override the first paragraph.
  @type x: type of keyword x, a string.
  @precondition: foo is not bar.
  @RETURN: return value.
  @postcondition: foo is bar.
  @invariant: baz does not qux.
  @type z: a type of a non-existing variable.
  @raise e: raises exception out-of-memory.
  @type b: type of b, string.
  @status: buggy.
  @change: added status.
  @change: removed status.
  @permission: read
  @since: version 1.0.1.
  @permission: write
  @attention: an attention description.
  @keyword x: an accepted keyword parameter.
  @attention: another attention description.
  @type: a standalone type of property, no var name.
  @deprecated: do not use this.
  @author: Martin
  @organization: Kite
  @copyright: (c) Kite
  @copyright: other
  @bug: bug 2 description.
  @warning: warning 1 description.
  @license: mit
  @license: bsd
  @contact: x@y.z
  @group Tools: zip, zap, *_tool
  @sort: zip, zap
  @warning: warning 2 description.
  """
  return 1
