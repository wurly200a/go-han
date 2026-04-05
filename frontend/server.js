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

// Proxy endpoint for GET /api/users
app.get('/api/users', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_API_BASE}/users`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching users:', error.message);
    res.status(500).json({ error: 'Failed to fetch users from backend' });
  }
});

// Proxy endpoint for PUT /api/users/:user_id/roles
app.put('/api/users/:user_id/roles', async (req, res) => {
  try {
    const response = await axios.put(`${BACKEND_API_BASE}/users/${req.params.user_id}/roles`, req.body);
    res.json(response.data);
  } catch (error) {
    console.error('Error updating user roles:', error.message);
    res.status(500).json({ error: 'Failed to update user roles in backend' });
  }
});

// Proxy endpoint for GET /api/cook-schedules
app.get('/api/cook-schedules', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_API_BASE}/cook-schedules`, { params: req.query });
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching cook schedules:', error.message);
    res.status(500).json({ error: 'Failed to fetch cook schedules from backend' });
  }
});

// Proxy endpoint for PUT /api/cook-schedules
app.put('/api/cook-schedules', async (req, res) => {
  try {
    const response = await axios.put(`${BACKEND_API_BASE}/cook-schedules`, req.body);
    res.json(response.data);
  } catch (error) {
    console.error('Error updating cook schedules:', error.message);
    res.status(500).json({ error: 'Failed to update cook schedules in backend' });
  }
});

// Proxy endpoint for DELETE /api/cook-schedules
app.delete('/api/cook-schedules', async (req, res) => {
  try {
    const response = await axios.delete(`${BACKEND_API_BASE}/cook-schedules`, { params: req.query });
    res.json(response.data);
  } catch (error) {
    console.error('Error deleting cook schedules:', error.message);
    res.status(500).json({ error: 'Failed to delete cook schedules from backend' });
  }
});

// Proxy endpoint for GET /api/cook-default-schedules
app.get('/api/cook-default-schedules', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_API_BASE}/cook-default-schedules`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching cook default schedules:', error.message);
    res.status(500).json({ error: 'Failed to fetch cook default schedules from backend' });
  }
});

// Proxy endpoint for PUT /api/cook-default-schedules
app.put('/api/cook-default-schedules', async (req, res) => {
  try {
    const response = await axios.put(`${BACKEND_API_BASE}/cook-default-schedules`, req.body);
    res.json(response.data);
  } catch (error) {
    console.error('Error updating cook default schedules:', error.message);
    res.status(500).json({ error: 'Failed to update cook default schedules in backend' });
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
