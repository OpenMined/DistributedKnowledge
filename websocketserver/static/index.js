// index.js

// 1) COPY-TO-CLIPBOARD
const copyButton = document.getElementById("copy-button");
const commandField = document.getElementById("install-command");

copyButton.addEventListener("click", () => {
  navigator.clipboard.writeText(commandField.value)
    .then(() => {
      //alert("Command copied to clipboard!");
      UIkit.notification({message: 'Copied to clipboard!', pos: 'bottom-right', timeout: 1000})
    })
    .catch((err) => {
      console.error("Failed to copy:", err);
    });
});

// 2) CONNECTION GRAPH ANIMATION
const canvas = document.getElementById("graph-canvas");
if (canvas) {
  const ctx = canvas.getContext("2d");
  let w, h;

  function initCanvas() {
    w = canvas.width = canvas.offsetWidth;
    h = canvas.height = canvas.offsetHeight;
  }

  // On resize, reinitialize canvas size
  window.addEventListener("resize", initCanvas);
  initCanvas();

  // Updated: Use 4 times more nodes (80 nodes)
  const points = Array.from({ length: 80 }).map(() => ({
    x: Math.random() * w,
    y: Math.random() * h,
    vx: (Math.random() - 0.5) * 0.4,
    vy: (Math.random() - 0.5) * 0.4,
  }));

  function animate() {
    ctx.clearRect(0, 0, w, h);
    // Move and draw points
    points.forEach((p) => {
      p.x += p.vx;
      p.y += p.vy;
      // Bounce off edges
      if (p.x < 0 || p.x > w) p.vx *= -1;
      if (p.y < 0 || p.y > h) p.vy *= -1;

      // Draw point (circle)
      ctx.beginPath();
      ctx.arc(p.x, p.y, 3, 0, Math.PI * 2);
      ctx.fillStyle = "#FCFCFD"; //"#3B82F6";
      ctx.fill();
    });

    // Draw lines between nearby points
    for (let i = 0; i < points.length; i++) {
      for (let j = i + 1; j < points.length; j++) {
        const p1 = points[i];
        const p2 = points[j];
        const dist = Math.hypot(p2.x - p1.x, p2.y - p1.y);
        // If points are within a certain distance, draw a connecting line
        if (dist < 120) {
          ctx.strokeStyle = "rgba(252, 252, 253, 0.3)"; //"rgba(59, 130, 246, 0.3)"; // #3B82F6 with alpha
          ctx.beginPath();
          ctx.moveTo(p1.x, p1.y);
          ctx.lineTo(p2.x, p2.y);
          ctx.stroke();
        }
      }
    }

    requestAnimationFrame(animate);
  }

  animate();
}

