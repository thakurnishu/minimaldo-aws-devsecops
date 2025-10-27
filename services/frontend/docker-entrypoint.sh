#!/bin/sh

# Create or overwrite the env-config.js file
cat > /usr/share/nginx/html/env-config.js <<EOL
window._env_ = {
  REACT_APP_API_URL: '${REACT_APP_API_URL:-http://backend:8080/api}'
};
EOL

# Start Nginx
exec nginx -g 'daemon off;'
