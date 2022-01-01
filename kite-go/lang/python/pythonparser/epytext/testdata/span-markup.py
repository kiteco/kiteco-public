# Fails with unbalanced "{" and "}", as expected (markup cannot span blocks).
def example():
  """
  This is a B{first paragraph.

  This is a second} paragraph.
  """
  return 1
