const express = require('express');
const axios = require('axios');
const app = express();
const cors = require('cors');
const bodyParser = require('body-parser');
const request = require('supertest');

app.use(cors());
app.use(bodyParser.json());

const BACKEND_URL = 'http://backend:8080/api/meals';

app.get('/api/meals', async (req, res) => {
  try {
    const response = await axios.get(BACKEND_URL);
    res.json(response.data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to fetch meals from backend' });
  }
});

app.put('/api/meals/:userId', async (req, res) => {
  const { userId } = req.params;
  try {
    const response = await axios.put(`${BACKEND_URL}/${userId}`, req.body);
    res.json(response.data);
  } catch (error) {
    res.status(500).json({ error: 'Failed to update meal in backend' });
  }
});

if (require.main === module) {
  app.listen(3000, () => {
    console.log('Server running on port 3000');
  });
}

module.exports = app;

// Test cases
if (process.env.NODE_ENV === 'test') {
  describe('Meal API Tests', () => {
    it('should fetch all meals', async () => {
      const res = await request(app).get('/api/meals');
      expect(res.status).toBe(200);
      expect(Array.isArray(res.body)).toBe(true);
    });

    it('should update a meal', async () => {
      const res = await request(app)
        .put('/api/meals/1')
        .send({ date: '2024-02-04', lunch: false, dinner: false });
      expect(res.status).toBe(200);
      expect(res.body).toHaveProperty('message');
    });
  });
}
