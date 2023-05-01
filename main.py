from flask import Flask, request, abort
from linebot import LineBotApi, WebhookHandler
from linebot.exceptions import InvalidSignatureError
from linebot.models import MessageEvent, TextMessage, TextSendMessage, ImageMessage
from pyzbar.pyzbar import decode
from PIL import Image
import os
import configparser

app = Flask(__name__)

# Read channel access token and channel secret from the line_secret_key.txt 
config = configparser.ConfigParser()
config.read('line_secret_key.txt')
channel_access_token = config.get('line', 'channel_access_token')
channel_secret = config.get('line', 'channel_secret')

line_bot_api = LineBotApi(channel_access_token)
handler = WebhookHandler(channel_secret)

@app.route("/callback", methods=['POST'])
def callback():
    # Get X-Line-Signature header value
    signature = request.headers['X-Line-Signature']

    # Get request body as text
    body = request.get_data(as_text=True)

    # Handle webhook body
    try:
        handler.handle(body, signature)
    except InvalidSignatureError:
        abort(400)

    return 'OK'

@handler.add(MessageEvent, message=TextMessage)
def handle_text_message(event):
    if event.message.text == '/join':
        user_id = event.source.user_id
        line_bot_api.reply_message(
            event.reply_token,
            TextSendMessage(text=f"Welcome! Your user_id: {user_id} has been stored as a member.")
        )

@handler.add(MessageEvent, message=ImageMessage)
def handle_image_message(event):
    message_content = line_bot_api.get_message_content(event.message.id)
    image = Image.open(message_content.content)

    decoded_data = decode(image)
    if decoded_data:
        qr_code_data = decoded_data[0].data.decode("utf-8")
        line_bot_api.reply_message(
            event.reply_token,
            TextSendMessage(text=f"QR Code Data: {qr_code_data}")
        )
    else:
        line_bot_api.reply_message(
            event.reply_token,
            TextSendMessage(text="Unable to decode the QR code. Please try again with a clearer image.")
        )

if __name__ == "__main__":
    app.run(debug=True, port=int(os.environ.get('PORT', 5000)))
