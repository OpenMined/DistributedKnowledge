# Running Your Own Network Server

This tutorial will guide you through setting up and running your own Distributed Knowledge network server. By hosting your own server, you can create a private network for your organization or community.

## Prerequisites

Before starting, ensure you have:

- A server with a public IP address or domain name
- Basic knowledge of networking and server administration
- SSL/TLS certificates for secure connections
- Go 1.x installed on your server

## Step 1: Clone the Repository

Start by cloning the Distributed Knowledge repository:

```bash
git clone https://github.com/OpenMined/DistributedKnowledge.git
cd DistributedKnowledge
```

## Step 2: Configure SSL/TLS Certificates

The WebSocket server requires SSL/TLS certificates for secure connections. You can use certificates from Let's Encrypt or any other certificate authority.

Place your certificates in the `websocketserver` directory:

```bash
cp /path/to/your/fullchain.pem websocketserver/server.crt
cp /path/to/your/privkey.pem websocketserver/server.key
```

## Step 3: Customize Server Configuration

Create a configuration file for the WebSocket server. Create a file named `config.json` in the `websocketserver/config` directory:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "cert_file": "./server.crt",
    "key_file": "./server.key"
  },
  "metrics": {
    "enabled": true,
    "persist": true,
    "persist_interval": 3600
  },
  "auth": {
    "required": true,
    "allow_anonymous": false,
    "user_timeout": 3600
  },
  "rate_limit": {
    "enabled": true,
    "requests_per_minute": 60,
    "burst": 10
  }
}
```

Adjust these settings as needed for your environment:

- **host**: Set to `0.0.0.0` to listen on all interfaces, or specify a particular IP
- **port**: The port on which the server will listen (default: 8080)
- **cert_file/key_file**: Paths to your SSL certificate and key
- **metrics**: Configuration for server metrics collection
- **auth**: Authentication settings
- **rate_limit**: Rate limiting to prevent abuse

## Step 4: Build the Server

Navigate to the WebSocket server directory and build the server:

```bash
cd websocketserver
go build
```

This will create an executable file named `websocketserver` in the current directory.

## Step 5: Run the Server

Start the WebSocket server:

```bash
./websocketserver
```

You should see output indicating that the server has started and is listening for connections.

For production use, you might want to run the server as a service. Here's an example systemd service file:

```ini
[Unit]
Description=Distributed Knowledge WebSocket Server
After=network.target

[Service]
Type=simple
User=dkuser
WorkingDirectory=/path/to/DistributedKnowledge/websocketserver
ExecStart=/path/to/DistributedKnowledge/websocketserver/websocketserver
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Save this file as `/etc/systemd/system/dkserver.service`, then enable and start the service:

```bash
sudo systemctl enable dkserver
sudo systemctl start dkserver
```

## Step 6: Configure Firewall

Ensure your firewall allows connections to the server port (default: 8080):

```bash
# For UFW (Ubuntu)
sudo ufw allow 8080/tcp

# For firewalld (CentOS/RHEL)
sudo firewall-cmd --zone=public --add-port=8080/tcp --permanent
sudo firewall-cmd --reload
```

## Step 7: Set Up DNS (Optional)

For a production server, you'll likely want a domain name pointing to your server. Configure your DNS settings to point a domain or subdomain to your server's IP address.

## Step 8: Test the Server

You can test that your server is running correctly by connecting to it with a WebSocket client:

```bash
# Using websocat tool
websocat wss://your-server-domain:8080
```

## Step 9: Configure Clients to Connect

Now that your server is running, you need to configure Distributed Knowledge clients to connect to it:

```bash
./dk -userId="user1" \
     -server="wss://your-server-domain:8080" \
     -modelConfig="./config/model_config.json" \
     -rag_sources="./data/knowledge_base.jsonl"
```

Make sure to distribute the server URL to all users who need to connect to your network.

## Step 10: Monitor Server Activity

The WebSocket server includes built-in metrics and monitoring. You can view these metrics by accessing:

```
https://your-server-domain:8080/metrics
```

For more detailed monitoring, you can integrate with tools like Prometheus and Grafana.

## Server Administration

### User Management

The server keeps track of active users. You can view them using:

```bash
curl -k https://your-server-domain:8080/api/users
```

### Server Logs

Check the server logs for troubleshooting:

```bash
sudo journalctl -u dkserver
```

### Restarting the Server

If you need to restart the server:

```bash
sudo systemctl restart dkserver
```

### Updating the Server

To update the server with a new version:

```bash
cd /path/to/DistributedKnowledge
git pull
cd websocketserver
go build
sudo systemctl restart dkserver
```

## Advanced Server Configuration

### Load Balancing

For high-traffic deployments, you can set up multiple server instances behind a load balancer like Nginx:

```nginx
upstream dkservers {
    server backend1.example.com:8080;
    server backend2.example.com:8080;
    server backend3.example.com:8080;
}

server {
    listen 443 ssl;
    server_name dknetwork.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass https://dkservers;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Database Configuration

The WebSocket server can use different database backends for persistence:

```json
{
  "database": {
    "type": "postgres",
    "connection_string": "host=localhost port=5432 user=dkuser password=mypassword dbname=dkserver",
    "max_connections": 10
  }
}
```

### Access Control

Implement IP-based access control by modifying your server configuration:

```json
{
  "auth": {
    "required": true,
    "allow_anonymous": false,
    "allowed_ips": ["192.168.1.0/24", "10.0.0.5"],
    "blocked_ips": ["203.0.113.0/24"]
  }
}
```

## Next Steps

Now that your server is up and running, consider:

1. Creating a [backup and recovery plan](../how-to-guides/server_backup.md)
2. Setting up [monitoring and alerting](../how-to-guides/server_monitoring.md)
3. Establishing [network policies](../how-to-guides/network_policies.md)
4. Scaling your server for [high availability](../how-to-guides/high_availability.md)

Running your own Distributed Knowledge server gives you complete control over your network infrastructure and enables private collaboration within your organization or community.