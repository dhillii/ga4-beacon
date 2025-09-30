# Google Analytics Beacon GA4 (Updated 2025)

Sometimes it is impossible to embed the JavaScript tracking code provided by Google Analytics: the host page does not allow arbitrary JavaScript, and there is no Google Analytics integration. However, not all is lost! **If you can embed a simple image (pixel tracker), then you can beacon data to Google Analytics.** This updated version supports Google Analytics 4 (GA4) using the Measurement Protocol v2.

For a great, hands-on explanation of how this works, check out the following guides:

* [Using a Beacon Image for GitHub, Website and Email Analytics](http://www.sitepoint.com/using-beacon-image-github-website-email-analytics/)
* [Tracking Google Sheet views with Google Analytics using GA Beacon](http://mashe.hawksey.info/2014/02/tracking-google-sheet-views-with-google-analytics/)

## New GA4 Features

This updated beacon service includes several enhancements:

- **GA4 Measurement Protocol v2 Support**: Full compatibility with Google Analytics 4
- **Configuration File Support**: API secrets and measurement IDs are stored securely in config files
- **Enhanced Event Tracking**: Sends structured events with custom parameters
- **Session Management**: Proper session ID generation and tracking
- **Custom Parameter Support**: Add custom tracking parameters via query strings
- **Anonymous User Tracking**: Designed for tracking public content without user identification

### Can I use this in production?

 If you intend to use this in production for your application, you should deploy **your own instance** of this service, which will allow you to scale the service up and down to meet your capacity needs, introspect the logs, customize the code, and so on.

Deploying your own instance is straightforward: fork this repo, configure your GA4 credentials, and deploy to your preferred platform (Google App Engine, Cloud Run, or any Go-compatible hosting service).

## Setup Instructions

### 1. Set up Google Analytics 4

First, log in to your Google Analytics account and [set up a new GA4 property](https://support.google.com/analytics/answer/9304153):

* Create a new GA4 property
* **Property name:** anything you want (e.g. "Beacon Tracking")
* **Data stream:** Web stream with your beacon service URL
* Copy the **Measurement ID** (format: `G-XXXXXXXXXX`)
* Generate an **API Secret** in Admin > Data Streams > [your stream] > Measurement Protocol > Create

### 2. Configure the Beacon Service

Create a `config.json` file with your GA4 credentials:

```json
{
  "measurement_id": "G-XXXXXXXXXX",
  "api_secret": "your-api-secret-here"
}
```

**Security Note**: Never commit your API secret to version control. Use environment-specific config files or environment variables.

### 3. Deploy Your Instance

**Option A: Local Development**
```bash
go run main.go
# Service runs on http://localhost:8080
```

**Option B: Google App Engine**
```bash
gcloud app deploy
```

**Option C: Docker**
```bash
docker build -t ga-beacon .
docker run -p 8080:8080 -v $(pwd)/config.json:/app/config.json ga-beacon
```

### 4. Add Tracking to Your Content

Add a tracking image to the pages you want to track:

* _https://your-beacon-service.com/account-name/page-path_
* `account-name` can be any identifier for grouping your tracking
* `page-path` is an arbitrary path that will appear in your GA4 reports

Example tracker markup if you are using Markdown:

```markdown
[![Analytics](https://your-beacon-service.com/my-project/welcome-page)](https://github.com/your-username/your-repo)
```

Or HTML:

```html
<img src="https://your-beacon-service.com/my-project/welcome-page" alt="Analytics" />
```

## Advanced Usage

### Pixel Tracking (Invisible)

For invisible tracking, append `?pixel` to the image URL:

```markdown
![Analytics](https://your-beacon-service.com/my-project/welcome-page?pixel)
```

### Badge Styles

Different badge styles are available:
- Default: SVG badge
- `?gif` - GIF badge
- `?flat` - Flat SVG badge  
- `?flat-gif` - Flat GIF badge

### Custom Parameters

Add custom tracking data via query parameters:

```
https://your-beacon-service.com/my-project/welcome-page?pixel&custom_source=newsletter&custom_campaign=launch
```

Custom parameters will be prefixed with `custom_` in GA4 events.

### Auto-Referer Tracking

Use the referer header for automatic path detection:

```
https://your-beacon-service.com/my-project/auto?pixel&useReferer
```


## Configuration Options

### Environment Variables

- `CONFIG_FILE`: Path to config file (default: `config.json`)
- `PORT`: Server port (default: `8080`)

### Config File Format

```json
{
  "measurement_id": "G-XXXXXXXXXX",
  "api_secret": "your-api-secret-here"
}
```

## GA4 Event Structure

The beacon sends `page_view` events to GA4 with the following parameters:

- `session_id`: Generated timestamp-based session ID
- `user_agent`: Browser user agent
- `ip_address`: Client IP address
- `timestamp`: Event timestamp in RFC3339 format
- `custom_*`: Any additional query parameters

## FAQ

- **How does this work?** Google Analytics 4 provides a [Measurement Protocol v2](https://developers.google.com/analytics/devguides/collection/protocol/ga4) which allows us to POST event data directly to Google servers. GA Beacon generates unique client IDs, manages sessions, and sends structured events to GA4 when the tracking image is requested.

- **Why do we need to proxy?** GA4's Measurement Protocol requires an API secret for authentication and proper event structure formatting. The beacon service handles client ID generation, session management, and event formatting that can't be done with a simple GET request.

- **What about referrals and other visitor information?** The static tracking pixel approach limits the information we can collect. However, this GA4 version captures more data than the original, including user agent, IP address, referer information (when available), and custom parameters passed via query strings.

- **Do I have to use the demo instance?** No, and you shouldn't for production use. Deploy your own instance for better reliability, security, and customization. The project is under MIT license.

- **Can I use this to track visits to my GitHub README?** No, GitHub blocks external tracking pixels for security reasons - see [this commit](https://github.com/igrigorik/ga-beacon/commit/6acd8627bb7be36f24f5516e9873c92719a50e55) for details.

- **Is this GDPR compliant?** The beacon tracks anonymous users without personal identification. However, IP addresses are collected, so you should review your privacy policy and consider implementing IP anonymization if required for your use case.

- **How do I view the data in GA4?** Events appear in GA4 under Events > All Events as `page_view` events. You can create custom reports and audiences based on the custom parameters you send.

## Migration from Universal Analytics

If you're migrating from the original GA Beacon (Universal Analytics), you'll need to:

1. Create a new GA4 property
2. Update your beacon URLs to point to your new GA4-enabled instance
3. Generate API secrets for the Measurement Protocol
4. Update your configuration with GA4 credentials

The tracking URLs remain compatible, but the backend now sends data to GA4 instead of Universal Analytics.