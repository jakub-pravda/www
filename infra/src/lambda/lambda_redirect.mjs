export const handler = async (event) => {
    const request = event.Records[0].cf.request;
    const host = request.headers.host[0].value;

    if (!host.startsWith('www.')) {
        const response = {
            status: '301',
            statusDescription: 'Moved Permanently',
            headers: {
                location: [{
                    key: 'Location',
                    value: `https://www.${host}${request.uri}`
                }]
            }
        };
        return response;
    }

    return request;
};