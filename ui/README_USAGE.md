# Harbor Token Broker UI

A React-based user interface for managing the Harbor Token Broker, including access log viewing and policy management.

## Features

- **Access Logs**: View token request history with filtering and pagination
- **Policy Management**: Create, edit, and delete authorization policies via UI
- **Real-time Updates**: Automatically refreshes data from the backend

## Development

### Prerequisites

- Node.js 20 or later
- npm or yarn

### Setup

```bash
cd ui
npm install
```

### Development Server

```bash
npm run dev
```

The UI will be available at http://localhost:5173 by default.

To connect to a backend running on a different port, set the `VITE_API_URL` environment variable:

```bash
VITE_API_URL=http://localhost:8080 npm run dev
```

### Building for Production

```bash
npm run build
```

The build output will be in the `dist` directory, which is served by the Go backend.

## Components

- **AccessLogs**: View and filter access logs
- **Policies**: Manage authorization policies
- **Card, Button, Input**: Reusable UI components based on shadcn/ui design

## Technology Stack

- React 18
- TypeScript
- Vite
- Tailwind CSS
- React Router
- Lucide Icons
