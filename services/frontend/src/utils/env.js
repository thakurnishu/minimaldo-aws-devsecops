// This file provides access to environment variables at runtime
export const getEnv = (key, defaultValue = undefined) => {
  // First try to get from window._env_ (runtime)
  if (window._env_ && window._env_[key] !== undefined) {
    return window._env_[key];
  }
  // Fall back to build-time environment variables
  return process.env[key] || defaultValue;
};

// Common environment variables
export const API_URL = getEnv('REACT_APP_API_URL');
