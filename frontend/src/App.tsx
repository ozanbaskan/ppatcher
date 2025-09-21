import { useEffect, useState } from "react";
import logo from "./assets/images/logo.jpeg";
import { EventsOn, EventsOff, EventsEmit } from "../wailsjs/runtime/runtime";
import { ManualUpdate, Config, StartExecutable } from "../wailsjs/go/main/App";

type DownloadStatus =
  | "idle"
  | "checking"
  | "downloading"
  | "ready"
  | "error"
  | "alreadyReady";

const DownloadStatusMapping: { [key in DownloadStatus]: string } = {
  idle: "",
  checking: "Checking for updates...",
  downloading: "Downloading files...",
  ready: "Ready",
  error: "Error occurred during update",
  alreadyReady: "Your files are up to date",
};

type ColorPaletteKey = "neutral" | "blue" | "green" | "purple";
type ColorPalette = {
  primary: string;
  primaryHover: string;
  secondary: string;
  secondaryHover: string;
  background: string;
  textPrimary: string;
  textSecondary: string;
  success: string;
  error: string;
  info: string;
  cardBg: string;
  progressBg: string;
  disabled: string;
  disabledText: string;
};

// Color palettes
const COLOR_PALETTES: Record<ColorPaletteKey, ColorPalette> = {
  neutral: {
    primary: "#6b7280",
    primaryHover: "#4b5563",
    secondary: "#9ca3af",
    secondaryHover: "#6b7280",
    background: "linear-gradient(135deg, #f9fafb 0%, #f3f4f6 100%)",
    textPrimary: "#374151",
    textSecondary: "#6b7280",
    success: "#059669",
    error: "#dc2626",
    info: "#2563eb",
    cardBg: "#ffffff",
    progressBg: "#e5e7eb",
    disabled: "#d1d5db",
    disabledText: "#9ca3af",
  },
  blue: {
    primary: "#3182ce",
    primaryHover: "#2b6cb0",
    secondary: "#63b3ed",
    secondaryHover: "#4299e1",
    background: "linear-gradient(135deg, #ebf8ff 0%, #bee3f8 100%)",
    textPrimary: "#2a4365",
    textSecondary: "#4c51bf",
    success: "#38a169",
    error: "#e53e3e",
    info: "#3182ce",
    cardBg: "#ffffff",
    progressBg: "#e6fffa",
    disabled: "#93c5fd",
    disabledText: "#93c5fd",
  },
  green: {
    primary: "#38a169",
    primaryHover: "#2f855a",
    secondary: "#68d391",
    secondaryHover: "#48bb78",
    background: "linear-gradient(135deg, #f0fff4 0%, #c6f6d5 100%)",
    textPrimary: "#22543d",
    textSecondary: "#38a169",
    success: "#38a169",
    error: "#e53e3e",
    info: "#3182ce",
    cardBg: "#ffffff",
    progressBg: "#e6fffa",
    disabled: "#9ae6b4",
    disabledText: "#9ae6b4",
  },
  purple: {
    primary: "#805ad5",
    primaryHover: "#6b46c1",
    secondary: "#9f7aea",
    secondaryHover: "#805ad5",
    background: "linear-gradient(135deg, #faf5ff 0%, #e9d8fd 100%)",
    textPrimary: "#322659",
    textSecondary: "#553c9a",
    success: "#38a169",
    error: "#e53e3e",
    info: "#3182ce",
    cardBg: "#ffffff",
    progressBg: "#e6fffa",
    disabled: "#d6bcfa",
    disabledText: "#d6bcfa",
  },
};

// Keyframes for animations
const keyframes = `
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
  
  @keyframes pulse {
    0% { transform: scale(1); }
    50% { transform: scale(1.05); }
    100% { transform: scale(1); }
  }
  
  @keyframes buttonClick {
    0% { transform: scale(1); }
    50% { transform: scale(0.95); }
    100% { transform: scale(1); }
  }
  
  @keyframes progressPulse {
    0% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0.4); }
    70% { box-shadow: 0 0 0 6px rgba(99, 102, 241, 0); }
    100% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0); }
  }
`;

// Inject keyframes into the document
const styleSheet = document.createElement("style");
styleSheet.innerText = keyframes;
document.head.appendChild(styleSheet);

function App() {
  const [config, setConfig] = useState({
    colorPalette: "neutral",
    showStartButton: false,
  });
  const [progress, setProgress] = useState(() => 0);
  const [downloadState, setDownloadState] = useState<DownloadStatus>("idle");
  const [isStartHovered, setIsStartHovered] = useState(false);
  const [isUpdateHovered, setIsUpdateHovered] = useState(false);
  const [isCheckButtonClicked, setIsCheckButtonClicked] = useState(false);
  const [isStartButtonClicked, setIsStartButtonClicked] = useState(false);
  const [statusKey, setStatusKey] = useState(0);
  const [isCheckButtonDisabled, setIsCheckButtonDisabled] = useState(false);
  const [isStartButtonDisabled, setIsStartButtonDisabled] = useState(false);

  // Get the current color palette
  const colors = COLOR_PALETTES[config.colorPalette as ColorPaletteKey];

  // Calculate responsive dimensions
  const windowHeight = window.innerHeight;
  const progressBarWidth = Math.min(window.innerWidth * 0.7, 500);
  const imageHeight = Math.min(280, windowHeight * 0.3);

  useEffect(() => {
    Config()
      .then((config) => {
        setConfig({
          colorPalette: config.colorPalette,
          showStartButton: !!config.executable,
        });
      })
      .catch(() => {});

    EventsOn("downloadStatus", (newStatus: DownloadStatus) => {
      // Trigger status animation by updating the key
      setStatusKey((prevKey) => prevKey + 1);

      setDownloadState((oldStatus) => {
        if (newStatus === "ready" || newStatus === "alreadyReady") {
          setProgress(() => 1);
        }
        if (
          (oldStatus === "ready" || oldStatus === "alreadyReady") &&
          newStatus === "ready"
        ) {
          return "alreadyReady";
        }

        return newStatus;
      });
    });

    EventsOn("downloadProgress", (progress: number) => {
      const normalizedProgress = Math.min(Math.max(progress, 0), 1);
      setProgress(() => normalizedProgress);
    });

    EventsEmit("ready");

    return () => {
      EventsOff("downloadStatus");
      EventsOff("downloadProgress");
    };
  }, []);

  const onUpdateClick = (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => {
    setIsStartButtonDisabled(true);
    setIsCheckButtonDisabled(true);
    setIsCheckButtonClicked(true);
    setTimeout(() => setIsCheckButtonClicked(false), 200);

    ManualUpdate().finally(() => {
      setTimeout(() => {
        setIsCheckButtonDisabled(false);
        setIsStartButtonDisabled(false);
      }, 200);
    });
  };

  const onStartClick = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    setIsStartButtonDisabled(true);
    setIsCheckButtonDisabled(true);

    setIsStartButtonClicked(true);
    setTimeout(() => setIsStartButtonClicked(false), 200);

    StartExecutable().finally(() => {
      setTimeout(() => {
        setIsStartButtonDisabled(false);
        setIsCheckButtonDisabled(false);
      }, 200);
    });
  };

  // Determine status color based on state
  const getStatusColor = () => {
    switch (downloadState) {
      case "ready":
        return colors.success;
      case "error":
        return colors.error;
      case "alreadyReady":
        return colors.info;
      default:
        return colors.textSecondary;
    }
  };

  return (
    <div
      id="App"
      style={{
        ...styles.app,
        background: colors.background,
        height: "100vh",
      }}
    >
      <div style={styles.content}>
        <div style={styles.header}>
          <h1 style={{ ...styles.title, color: colors.textPrimary }}>
            PPatcher
          </h1>
          <p style={{ ...styles.subtitle, color: colors.textSecondary }}>
            Keep your files up to date
          </p>
        </div>

        <div
          style={{ ...styles.logoContainer, backgroundColor: colors.cardBg }}
        >
          <img
            src={logo}
            alt="PPatcher Logo"
            style={{ ...styles.logo, height: imageHeight }}
          />
        </div>

        <div style={styles.statusContainer}>
          <div
            key={statusKey}
            id="status"
            style={{
              ...styles.statusText,
              color: getStatusColor(),
              animation: "fadeIn 0.5s ease-out",
              minHeight: "24px",
            }}
          >
            {DownloadStatusMapping[downloadState]}
          </div>

          {downloadState === "downloading" && (
            <div style={{ ...styles.progressText, color: colors.primary }}>
              {Math.round(progress * 100)}%
            </div>
          )}
        </div>

        <div style={styles.progressContainer}>
          <div
            style={{
              ...styles.progressBar,
              width: `${progressBarWidth}px`,
              backgroundColor: colors.progressBg,
            }}
          >
            <div
              style={{
                ...styles.progressFill,
                width: `${progress * 100}%`,
                backgroundColor:
                  downloadState === "error"
                    ? colors.error
                    : downloadState === "ready" ||
                      downloadState === "alreadyReady"
                    ? colors.success
                    : colors.primary,
                animation:
                  downloadState === "downloading"
                    ? "progressPulse 2s infinite"
                    : "none",
              }}
            ></div>
          </div>
        </div>

        <div
          style={{
            ...styles.buttonContainer,
            justifyContent: config.showStartButton ? "center" : "center",
          }}
        >
          {config.showStartButton && (
            <button
              onClick={onStartClick}
              disabled={isStartButtonDisabled}
              onMouseEnter={() =>
                !isStartButtonDisabled && setIsStartHovered(true)
              }
              onMouseLeave={() => setIsStartHovered(false)}
              style={{
                ...styles.button,
                backgroundColor: isStartButtonDisabled
                  ? colors.disabled
                  : colors.primary,
                color: isStartButtonDisabled ? colors.disabledText : "white",
                animation: isStartButtonClicked
                  ? "buttonClick 0.2s ease"
                  : "none",
                cursor: isStartButtonDisabled ? "not-allowed" : "pointer",
                ...(!isStartButtonDisabled &&
                  isStartHovered && {
                    backgroundColor: colors.primaryHover,
                    transform: "translateY(-2px)",
                    boxShadow: "0 4px 8px rgba(0, 0, 0, 0.12)",
                  }),
              }}
            >
              Start
            </button>
          )}
          <button
            onClick={onUpdateClick}
            disabled={isCheckButtonDisabled}
            onMouseEnter={() =>
              !isCheckButtonDisabled && setIsUpdateHovered(true)
            }
            onMouseLeave={() => setIsUpdateHovered(false)}
            style={{
              ...styles.button,
              backgroundColor: isCheckButtonDisabled
                ? colors.disabled
                : colors.primary,
              color: isCheckButtonDisabled ? colors.disabledText : "white",
              animation: isCheckButtonClicked
                ? "buttonClick 0.2s ease"
                : "none",
              cursor: isCheckButtonDisabled ? "not-allowed" : "pointer",
              ...(!isCheckButtonDisabled &&
                isUpdateHovered && {
                  backgroundColor: colors.primaryHover,
                  transform: "translateY(-2px)",
                  boxShadow: "0 4px 8px rgba(0, 0, 0, 0.12)",
                }),
            }}
          >
            Check for Updates
          </button>
        </div>
      </div>

      <div style={styles.footer}>
        <p style={{ ...styles.footerText, color: colors.textSecondary }}>
          PPatcher v1.0.0
        </p>
      </div>
    </div>
  );
}

const styles = {
  app: {
    display: "flex",
    flexDirection: "column" as "column",
    alignItems: "center",
    justifyContent: "space-between",
    fontFamily:
      "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif",
    padding: "20px",
    boxSizing: "border-box" as "border-box",
    overflow: "hidden",
  },
  content: {
    display: "flex",
    flexDirection: "column" as "column",
    alignItems: "center",
    width: "100%",
    flex: "1",
    justifyContent: "center",
    gap: "20px",
  },
  header: {
    textAlign: "center" as "center",
  },
  title: {
    fontSize: "2rem",
    fontWeight: "700",
    margin: "0 0 6px 0",
    letterSpacing: "-0.5px",
  },
  subtitle: {
    fontSize: "0.9rem",
    margin: 0,
    fontWeight: "400",
  },
  logoContainer: {
    padding: "12px",
    borderRadius: "12px",
    boxShadow: "0 4px 12px rgba(0, 0, 0, 0.05)",
  },
  logo: {
    borderRadius: "8px",
    display: "block",
  },
  statusContainer: {
    display: "flex",
    flexDirection: "column" as "column",
    alignItems: "center",
    minHeight: "50px",
    justifyContent: "center",
    gap: "8px",
  },
  statusText: {
    fontSize: "1rem",
    fontWeight: "500",
    textAlign: "center" as "center",
  },
  progressText: {
    fontSize: "1.2rem",
    fontWeight: "600",
  },
  progressContainer: {
    display: "flex",
    justifyContent: "center",
    width: "100%",
  },
  progressBar: {
    height: "16px",
    borderRadius: "8px",
    overflow: "hidden",
  },
  progressFill: {
    height: "100%",
    borderRadius: "8px",
    transition: "width 0.3s ease",
  },
  buttonContainer: {
    display: "flex",
    gap: "12px",
    width: "100%",
    maxWidth: "320px",
  },
  button: {
    padding: "12px 20px",
    borderRadius: "8px",
    border: "none",
    fontSize: "0.95rem",
    fontWeight: "500",
    transition: "all 0.2s ease",
    flex: "1",
    minWidth: "140px",
    boxShadow: "0 2px 6px rgba(0, 0, 0, 0.1)",
  },
  footer: {
    padding: "12px",
  },
  footerText: {
    fontSize: "0.75rem",
    margin: 0,
  },
};

export default App;
