import unittest
from unittest.mock import AsyncMock, MagicMock
import asyncio
from function import func


class TestFunction(unittest.TestCase):
    def setUp(self):
        self.function = func.new()

    def test_handle_echo(self):
        """Test that the function echoes query parameters"""
        async def test():
            # Mock ASGI scope with a query parameter
            scope = {
                'type': 'http',
                'query_string': b'test-echo-param=hello',
                'path': '/',
                'headers': []
            }
            
            receive = AsyncMock()
            responses = []
            
            async def send(message):
                responses.append(message)
            
            await self.function.handle(scope, receive, send)
            
            # Check that we got the expected response
            self.assertEqual(len(responses), 2)
            self.assertEqual(responses[0]['status'], 200)
            self.assertEqual(responses[1]['body'], b'hello')
        
        asyncio.run(test())

    def test_handle_no_query(self):
        """Test that the function returns OK when no query parameters"""
        async def test():
            # Mock ASGI scope without query parameters
            scope = {
                'type': 'http',
                'query_string': b'',
                'path': '/',
                'headers': []
            }
            
            receive = AsyncMock()
            responses = []
            
            async def send(message):
                responses.append(message)
            
            await self.function.handle(scope, receive, send)
            
            # Check that we got the expected response
            self.assertEqual(len(responses), 2)
            self.assertEqual(responses[0]['status'], 200)
            self.assertEqual(responses[1]['body'], b'OK')
        
        asyncio.run(test())


if __name__ == '__main__':
    unittest.main()