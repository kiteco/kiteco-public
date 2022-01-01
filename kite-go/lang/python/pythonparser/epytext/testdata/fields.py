# epydoc recognizes @param f2 as a parameter despite the space after the
# argument, but does not recognize note as a field (because of the space).
def sample(f1, f2, f3, f4):
  """
  @see: field 1
  @note : is it a field? has space before colon
  @param f1: field 3 with an arg
  @type f1: integer
  @param f2 : is it a field? has space before colon
  @return: some value
  @param f3: another one
  """
  return 1
