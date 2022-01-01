export default Object.freeze({
  ZEROES: {
    SLASHED: 'slashed',
    HOLLOW: 'hollow',
    DOTTED: 'dotted'
  },
  FONTS: [
    {
      value: '1',
      name: 'Input',
      zeroes: {
        slashed: true,
        hollow: true,
        dotted: true
      },
      description: 'Modern sans/serif power duo'
    },
    {
      value: '2',
      name: 'Monoid',
      zeroes: {
        slashed: true,
        hollow: true,
        dotted: true
      },
      description: 'Customizable, slender, open source'
    },
    {
      value: '3',
      name: 'Hack',
      zeroes: {
        slashed: false,
        hollow: false,
        dotted: true
      },
      description: 'An open source workhorse for source code'
    },
    {
      value: '4',
      name: 'IBM VGA',
      zeroes: {
        slashed: false,
        hollow: false,
        dotted: true
      },
      description: 'For a particularly retro day'
    }
  ],
  THEMES: {
    LIGHT: 'light',
    DARK: 'dark'
  },
  PAGE_KIND: {
    HOWTO: 'howto',
    IDENTIFIER: 'identifier'
  }
})