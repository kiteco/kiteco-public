# If B{Two} is right after the period, it doesn't work. It needs a space
# before "B{". But for some reason I{Three} works even without a space
# after the preceding period.
#
# As expected B{Four} doesn't work with the newline in between.
# As expected, I{Five} works, and 6, 7 and 8 work too.
# 9 works too, so really, any markup letter right before { should
# work.
def example():
  """
  I{One}. B{Two}.I{
  Three}.B
  {Four}. I{Five}B{Six},I{Seven}:B{Eight}
  AlloB{Nine}.
  """
  return 1

