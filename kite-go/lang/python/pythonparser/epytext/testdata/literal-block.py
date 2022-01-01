# The first two are treated as literal blocks.
def example():
  """
  This is a paragraph::
    With a literal block.
  While this one has trailing space::
    So is this a literal block?
  And this one has trailing chars:: !
    - So this clearly isn't a literal block.
  """
  return 1
