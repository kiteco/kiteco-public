function main() {
  return People({
    user_selectors: [
      {
        selector: 'user["$email"] == "%s"'
      }
    ]
  });
}
