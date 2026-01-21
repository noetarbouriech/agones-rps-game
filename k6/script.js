import ws from "k6/ws";
import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: parseInt(__ENV.K6_VUS) || 100,
  duration: __ENV.K6_DURATION || "20s",
};

export default function () {
  const wsURL = "ws://localhost:3000/ws";

  const params = { tags: { test: "websocket-match" } };

  const res = ws.connect(wsURL, params, function (socket) {
    socket.on("open", function open() {
      console.log("WebSocket Connected");
    });

    socket.on("message", function message(data) {
      const matchURL = data.toString().trim();
      console.log(`Received match URL: ${matchURL}`);

      // Make HTTP GET request to the received URL
      const httpRes = http.get(matchURL);

      // Check if the HTTP request was successful
      check(httpRes, {
        "match URL status is 200": (r) => r.status === 200,
      });

      console.log(`HTTP Response status: ${httpRes.status}`);

      // Close WebSocket after processing
      socket.close();
    });

    socket.on("close", function close() {
      console.log("WebSocket disconnected");
    });

    socket.on("error", function error(err) {
      console.log("WebSocket error:", err);
    });
  });

  // Check WebSocket connection status
  check(res, {
    "websocket status is 101": (r) => r && r.status === 101,
  });
}
