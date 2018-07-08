from credentials.credentials_pb2 import GetSessionRequest
from credentials.credentials_pb2_grpc import CredentialsStub
import grpc


if __name__ == '__main__':
    channel = grpc.insecure_channel(target="localhost:8081")
    credentials = CredentialsStub(channel=channel)
    sess = credentials.GetSession(GetSessionRequest(userid="677c6781-a760-4baf-9269-1311454d34e3"))
    print(sess)
