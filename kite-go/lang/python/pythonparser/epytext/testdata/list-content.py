# Some things to note:
# 2. is not a list bullet, because it lacks a space, so it is part
#    of the 1st paragraph.
# 3. is indeed rendered as a literal block, note that there is a `:`
#    rendered after the bullet list (considered a paragraph?)
# 4. and 5. work as expected.
# 6. doesn't render as doctest.
# 7. and 8. work, doctest really needs blank lines before and after.
def example():
  """
  1. Standard paragraph.
  2.No space after bullet.
  3. ::
    Literal block?
  4.
  Paragraph on next line, unindented.
  5.
    Paragraph on next line, indented.

  6. >>> doctest on same list line
  Should not work?

  7.

    >>> doctest on next line
    should work?

  8. With a paragraph before...

    >>> doctest should work?
    maybe?

  """
  return 1
