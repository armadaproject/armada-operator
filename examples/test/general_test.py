from armada_client.client import ArmadaClient
import grpc

channel = grpc.insecure_channel(f"example.com:50051")

client = ArmadaClient(channel)

# Create Queue 

queue_req = client.create_queue_request(name='test', priority_factor=1.0)
client.create_queue(queue_req)
