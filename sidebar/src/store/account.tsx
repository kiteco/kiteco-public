enum AccountStates {
    ExistsNoPassword = "ExistsNoPasswordError",
    ExistsWithPassword = "ExistsWithPassword",
    IsNew = "NewAccount",
}

enum Errors {
    // InvalidEmail is determined by kited here
    InvalidEmailError = "InvalidEmail",
    NoEmailError = "NoEmailError",
    NotOnlineError = "NotOnlineError"
}

async function getAccountState(
  email: string,
  checkEmail: (email: string) => Promise<any>,
  forceCheckOnline: () => Promise<any>,
): Promise<{ state: string } | { error: string }> {
  const { success, isOnline } = await forceCheckOnline()
  if (success && !isOnline ) {
    return { error: Errors.NotOnlineError }
  }

  if (!email) {
    return { error: Errors.NoEmailError }
  }

  const { error } = await checkEmail(email)

  if (error && error.fail_reason === "invalid email address") {
    return { error: Errors.InvalidEmailError }
  }

  if (error) {
    const {
      account_exists: accountExists,
      has_password: hasPassword,
    } = error

    if (accountExists && !hasPassword) {
      return { state: AccountStates.ExistsNoPassword }
    } else if (accountExists && hasPassword) {
      return { state: AccountStates.ExistsWithPassword }
    } else {
      return { error: error.fail_reason }
    }
  }

  return { state: AccountStates.IsNew }
}

export {
  Errors,
  AccountStates,
  getAccountState,
}
