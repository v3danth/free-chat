const axios = require("axios");
const WebSocket = require("ws");

const BASE_URL = "http://localhost:8080";
const WS_URL = "ws://localhost:8080/ws";
const PASSWORD = "secret123";

const USERS = [
  "john@example.com",
  "john2@example.com",
  "john3@example.com",
];

async function login(email) {
  const res = await axios.post(`${BASE_URL}/auth/login`, {
    email,
    password: PASSWORD,
  });

  return {
    token: res.data.token,
    user: res.data.user,
  };
}

async function connectUser(email) {
  const { token, user } = await login(email);

  console.log(`Logged in: ${user.Username} (${user.ID})`);

  const ws = new WebSocket(`${WS_URL}?token=${token}`);

  return new Promise((resolve, reject) => {
    ws.on("open", () => {
      
      console.log(`WS connected: ${user.Username}`);
      ws.send(
        JSON.stringify({
          type: "join",
          room_id: 1,
        })
      );

      resolve({
        ws,
        user,
      });
    });

    ws.on("error", reject);
  });
}

async function run() {
  const clients = await Promise.all(
    USERS.map(connectUser)
  );

  clients.forEach(({ ws, user }) => {
    ws.on("message", (msg) => {
      console.log(
        `[RECV ${user.Username}] ${msg.toString()}`
      );
    });
  });

  setTimeout(() => {
    clients.forEach(({ ws, user }) => {
      ws.send(
        JSON.stringify({
          type: "chat",
          room_id: 1,
          content: `hello from ${user.Username}`,
        })
      );

      console.log(
        `[SEND ${user.Username}] hello from ${user.Username}`
      );
    });
  }, 1000);

  setTimeout(() => {
    console.log("\n=== SUMMARY ===");
    console.log(
      `Connected clients: ${clients.length}`
    );

    clients.forEach(({ ws }) => ws.close());

    process.exit(0);
  }, 5000);
}

run().catch(console.error);
