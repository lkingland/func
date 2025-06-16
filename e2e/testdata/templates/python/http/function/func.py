# Function
import logging
import urllib.parse


def new():
    """ New is the only method that must be implemented by a Function.
    The instance returned can be of any name.
    """
    return Function()


class Function:
    def __init__(self):
        """ The init method is an optional method where initialization can be
        performed. See the start method for a startup hook which includes
        configuration.
        """

    async def handle(self, scope, receive, send):
        """ Handle all HTTP requests to this Function other than readiness
        and liveness probes."""

        logging.info("Echo: Request Received")

        # Extract the full URL from the ASGI scope
        scheme = scope.get('scheme', 'http')
        headers_dict = {}
        for header_name, header_value in scope.get('headers', []):
            headers_dict[header_name.lower()] = header_value.decode('utf-8')
        
        # Get host from headers or use default
        host = headers_dict.get(b'host', 'localhost')
        if isinstance(host, bytes):
            host = host.decode('utf-8')
        
        path = scope.get('path', '/')
        query_string = scope.get('query_string', b'').decode('utf-8')
        
        # Construct the full URL
        full_url = f"{scheme}://{host}{path}"
        if query_string:
            full_url += f"?{query_string}"
        
        # The response body is the full URL
        response_body = full_url

        # Echo the request URL back to the client
        await send({
            'type': 'http.response.start',
            'status': 200,
            'headers': [
                [b'content-type', b'text/plain'],
            ],
        })
        await send({
            'type': 'http.response.body',
            'body': response_body.encode(),
        })

    def start(self, cfg):
        """ start is an optional method which is called when a new Function
        instance is started, such as when scaling up or during an update.
        Provided is a dictionary containing all environmental configuration.
        Args:
            cfg (Dict[str, str]): A dictionary containing environmental config.
                In most cases this will be a copy of os.environ, but it is
                best practice to use this cfg dict instead of os.environ.
        """
        logging.info("Function starting")

    def stop(self):
        """ stop is an optional method which is called when a function is
        stopped, such as when scaled down, updated, or manually canceled.  Stop
        can block while performing function shutdown/cleanup operations.  The
        process will eventually be killed if this method blocks beyond the
        platform's configured maximum studown timeout.
        """
        logging.info("Function stopping")

    def alive(self):
        """ alive is an optional method for performing a deep check on your
        Function's liveness.  If removed, the system will assume the function
        is ready if the process is running. This is exposed by default at the
        path /health/liveness.  The optional string return is a message.
        """
        return True, "Alive"

    def ready(self):
        """ ready is an optional method for performing a deep check on your
        Function's readiness.  If removed, the system will assume the function
        is ready if the process is running.  This is exposed by default at the
        path /health/readiness.
        """
        return True, "Ready"