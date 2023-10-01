const apiKey = 'AIzaSyDbe_27QgSa811d4VFf842xIdAkLIIZgYo';
const placeId = 'ChIJefayK7LxC0cRM64ybg9kg7Q'; // The Google Place ID of the shop

// Get the current day of the week (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
const currentDay = new Date().getDay() + 6;
console.log(`Today is ${currentDay}.`);

var request = {
  placeId: 'ChIJN1t_tDeuEmsRUsoyG83frY4',
  fields: ['name', 'opening_hours']
};

service = new google.maps.places.PlacesService(document.createElement('div'));
service.getDetails(request, callback)
.catch(error => console.error('Error fetching data:', error));;

function callback(place, status) {
  if (status == google.maps.places.PlacesServiceStatus.OK) {
    console.log(place);

    const openingHours = place.opening_hours;

    if (openingHours) {
      const todayOpeningHours = openingHours.periods
      .filter(period => period.open.day === currentDay)
      .map(openingHours => `${openingHours.open.time} - ${openingHours.close.time}`)
      .join(', ');

      console.log(todayOpeningHours);
      resultElement = document.getElementById('openinghours')
      if (todayOpeningHours.length > 0) {
        resultElement.textContent = `DNES OTEVŘENO: ${todayOpeningHours}`;
      } else {
        resultElement.textContent = `DNES MÁME ZAVŘENO`;
      }
    } else {
      console.log('Unable to retrieve opening hours.');
    }
  }
}
