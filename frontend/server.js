const express = require('express');
const axios = require('axios');
const path = require('path');
const bodyParser = require('body-parser');
const cors = require('cors');

const app = express();

// Middleware
app.use(bodyParser.json());
app.use(express.static(path.join(__dirname, 'public')));
app.use(cors());

// Set backend API base URL (can be overridden via environment variable)
const BACKEND_API_BASE = process.env.BACKEND_API_BASE || 'http://backend:8080/api';

// Health check endpoint for the frontend.
// This endpoint proxies the backend's /health endpoint.
app.get('/health', async (req, res) => {
  try {
    console.log('Frontend /health endpoint called');
    const response = await axios.get(`${BACKEND_API_BASE}/health`);
    console.log('Backend /health response:', response.data);
    res.json(response.data);
  } catch (error) {
    console.error('Error in frontend /health endpoint:', error.message);
    res.status(500).json({ error: 'Backend health check failed' });
  }
});

// Proxy endpoint for GET /api/meals
app.get('/api/meals', async (req, res) => {
  try {
    // Forward query parameters (date and days) to the backend
    const response = await axios.get(`${BACKEND_API_BASE}/meals`, { params: req.query });
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching meals:', error.message);
    res.status(500).json({ error: 'Failed to fetch meals from backend' });
  }
});

// Proxy endpoint for bulk update of meals.
app.put('/api/meals/bulk-update', async (req, res) => {
  try {
    const response = await axios.put(`${BACKEND_API_BASE}/meals/bulk-update`, req.body);
    res.json(response.data);
  } catch (error) {
    console.error('Error bulk updating meals:', error.message);
    res.status(500).json({ error: 'Failed to bulk update meals in backend' });
  }
});

// Proxy endpoint for GET /api/user-defaults/:user_id
app.get('/api/user-defaults/:user_id', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_API_BASE}/user-defaults/${req.params.user_id}`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching user defaults:', error.message);
    res.status(500).json({ error: 'Failed to fetch user defaults from backend' });
  }
});

// Proxy endpoint for PUT /api/user-defaults/:user_id
app.put('/api/user-defaults/:user_id', async (req, res) => {
  try {
    const response = await axios.put(`${BACKEND_API_BASE}/user-defaults/${req.params.user_id}`, req.body);
    res.json(response.data);
  } catch (error) {
    console.error('Error updating user defaults:', error.message);
    res.status(500).json({ error: 'Failed to update user defaults in backend' });
  }
});

// Serve index.html on the root path
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

// Start server
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Frontend server running on port ${PORT}`);
});
