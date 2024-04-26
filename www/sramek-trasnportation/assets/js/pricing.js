fetch('assets/pricing.json')
  .then(response => response.json())
  .then(data => {
    for (let key in data.landfill) {
      document.querySelector(`#${key}Name`).textContent = data.landfill[key].name;
      document.querySelector(`#${key}Price`).textContent = data.landfill[key].price;
    }
  })
  .catch(error => console.error('Error:', error));