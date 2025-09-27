# Network Port Manager

A simple terminal-based application to scan and manage active listening network ports on your system. This application is designed exclusively for macOS and Linux.

## Features

- Scans for processes listening on TCP ports
- Displays port number, process name, and PID
- Allows terminating selected processes
- Real-time refresh of port list

## Requirements

- Go 1.16 or later
- macOS or Linux (uses `lsof` command)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/queaxtra/oximex.git
   cd oximex
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Run the application:
   ```
   go run main.go
   ```

### Controls

- Arrow keys: Navigate the list
- r: Refresh port list
- k: Kill selected process
- q: Quit application

## Contributing

We welcome contributions! To contribute:

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Make your changes, ensuring code follows Go best practices.
4. Add or update tests as needed.
5. Submit a pull request with a clear description of your changes.

Please ensure your code adheres to the project's coding standards and includes appropriate documentation.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

## Contact

If you have any questions about the project, please send an email to `fatih@etik.com`.
