const corsHeaders = {
  "Access-Control-Allow-Origin": "https://www.sramek-autodoprava.cz",
  "Access-Control-Allow-Headers": "*",
  "Access-Control-Allow-Methods": "OPTIONS,POST",
}

export const handler = async (event) => {
  if (event.httpMethod === "OPTIONS") {    
    // Handle preflight request
    return {
      statusCode: 200,
      headers: corsHeaders,
      body: JSON.stringify({ message: "CORS preflight response" }),
    };
  } return {
    statusCode: 400,
    headers: corsHeaders,
  };
};