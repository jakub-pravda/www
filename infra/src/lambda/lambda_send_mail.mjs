import { SESClient, SendEmailCommand } from "@aws-sdk/client-ses";
const ses = new SESClient({ region: "eu-central-1" });
const corsHeaders = {
  "Access-Control-Allow-Origin": "https://www.sramek-autodoprava.cz, https://sramek-autodoprava.cz",
  "Access-Control-Allow-Headers": "*",
  "Access-Control-Allow-Methods": "OPTIONS,POST",
}

export const handler = async (event) => {
  // decode from base 64
  const decodedBody = atob(event.body);
  let body = JSON.parse(decodedBody);
  
  console.info("Form request" + body);

  const command = new SendEmailCommand({
    Destination: {
      ToAddresses: ["objednavky@sramek-autodoprava.cz"],
    },
    Message: {
      Body: {
        Text: { Data: `Name: ${body.name}\nEmail: ${body.email}\n\nMessage: ${body.message}` },
      },
      Subject: { Data: `Order from ${body.name}` },
    },
    Source: "form@sramek-autodoprava.cz",
  });

  try {
    let response = await ses.send(command);
    return {
      statusCode: 200,
      headers: corsHeaders,
      body: JSON.stringify({ message: "Email sent successfully", response }),
    };
  } catch (error) {
    console.error(error);
    return {
      statusCode: 500,
      headers: corsHeaders,
      body: JSON.stringify({ message: "Failed to send email", error }),
    };
  }
};