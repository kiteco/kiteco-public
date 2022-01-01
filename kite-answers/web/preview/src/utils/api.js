export const render = (component, request) => {
  fetch(request)
    .then(response => {
      if (!response.ok) {
        const err = Error();
        err.name = response.statusText;
        err.message = response.status;
        throw err;
      }
      return response.json();
    })
    .then(result => {
      component.setState({
        isLoaded: true,
        input: result
      });
    })
    .catch(error => {
      component.setState({
        isLoaded: true,
        error
      });
    });
};
