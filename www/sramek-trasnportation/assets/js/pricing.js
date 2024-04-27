fetch('assets/pricing.json')
  .then(response => response.json())
  .then(data => {
    for (let key in data.landfill) {
      document.querySelector(`#${key}-name`).textContent = data.landfill[key].name;
      document.querySelector(`#${key}-price`).textContent = data.landfill[key].price;
    }
  })
  .catch(error => console.error('Error:', error));