package com.mokshesh.lld;

import java.time.LocalDateTime;
import java.util.List;

/**
 * A parking lot is a designated area for parking vehicles and is a feature found in almost all popular venues such as shopping malls, sports stadiums, offices, etc.

In a parking lot, there are a fixed number of parking spots available for different types of vehicles. 

Each of these spots is charged according to the time the vehicle has been parked in the parking lot. 

The parking time is tracked with a parking ticket issued to the vehicle at the entrance of the parking lot. 

Once the vehicle is ready to exit, it can either pay at the automated exit panel or to the parking agent at the exit using a card or cash payment method.
 */
public class ParkingLotSystem {

    public enum VehicleType {
        CAR, BIKE
    }

    public abstract class Vehicle {
        String licensePlate;
        VehicleType type;
        Vehicle(String licensePlate, VehicleType type){
            this.licensePlate = licensePlate;
            this.type = type;
        }
        String getLicensePlate(){
            return licensePlate;
        }
        VehicleType getVehicleType() {
            return type;
        }        
    }
    
    class Car extends Vehicle {
        Car(String licensePlate){
            super(licensePlate, VehicleType.CAR);
        }        
    }
    
    class Bike extends Vehicle {
        Bike(String licensePlate){
            super(licensePlate, VehicleType.BIKE);
        }        
    }
    
    enum ParkingSpotType {
        SMALL, MEDIUM, LARGE
    }
    
    abstract class ParkingSpot {
        String spotId;        
        ParkingSpotType parkingSpotType;
        Vehicle vehicle;
        boolean isOccupied;
        
        ParkingSpot(String spotId, ParkingSpotType parkingSpotType){
            this.spotId = spotId;
            this.parkingSpotType = parkingSpotType;            
        }
        
        boolean isOccupied() {
            return isOccupied;
        }
        
        boolean park(Vehicle vehicle){
            if(!isOccupied){
                this.vehicle = vehicle;
                this.isOccupied = true;
                return true;
            }
            return false;
        }
        
        Vehicle unpark(){
            Vehicle vehicle = this.vehicle;
            isOccupied = false;
            this.vehicle = null;
            return vehicle;            
        }
        
        ParkingSpotType getSpotType(){
            return parkingSpotType;
        }
    }
    class SmallParkingSpot extends ParkingSpot {
        SmallParkingSpot(String spotId){
            super(spotId, ParkingSpotType.SMALL);
        }
    }
    
    class ParkingTicket {
        String ticketId;
        ParkingSpot parkingSpot;
        Vehicle vehicle;
        boolean isPaid;
        LocalDateTime entryDateTime;
        LocalDateTime exitDateTime;
        
        ParkingTicket(String ticketId, Vehicle vehicle, ParkingSpot parkingSpot){
            this.ticketId = ticketId;
            this.vehicle = vehicle;
            this.entryDateTime = LocalDateTime.now();
        }
        
        boolean isPaid(){
            return isPaid;
        }
        
        void markAsPaid(){
            this.isPaid = true;
            this.exitDateTime = LocalDateTime.now();
        }
        LocalDateTime getEntryDateTime(){
            return entryDateTime;
        }
        LocalDateTime getExitDateTime(){
            return exitDateTime;
        }
        VehicleType getVehicleType(){
            return vehicle.getVehicleType();
        }
        String getLicensePlate(){
            return vehicle.getLicensePlate();
        }
    }
    
    enum PaymentMethod{
        CARD, CASH, UPI
    }
    
    // Strategy Pattern
    public interface PricingStrategy {
        double calculatePrice(LocalDateTime entryTime, LocalDateTime exitTime, VehicleType vehicleType);
    }
    public class HourlyPricingStrategy implements PricingStrategy {
        public double calculatePrice(LocalDateTime entryTime, LocalDateTime exitTime, VehicleType vehicleType){
            return 0D;
        }
    }
    
    class PaymentProcessor {
        private PricingStrategy pricingStrategy;
    
        public PaymentProcessor(PricingStrategy pricingStrategy) {
            this.pricingStrategy = pricingStrategy;
        }
        
        public boolean processPayment(ParkingTicket ticket, PaymentMethod paymentMethod) {
            double amount = pricingStrategy.calculatePrice(ticket.getEntryDateTime(), LocalDateTime.now(), ticket.getVehicleType());
            boolean isPaymentProcessed = executePayment(amount, paymentMethod);
            if(isPaymentProcessed){
                ticket.markAsPaid();
            }
            return true;
        }
        private boolean executePayment(double amount, PaymentMethod paymentMethod) {
                // Implementation for actual payment processing
                // This would integrate with real payment systems
            return true;
        }
    }
    
    class ParkingLot {
        List<ParkingSpot> parkingSpots;
        private static ParkingLot instance = null;
        public static ParkingLot getInstance(){
            if (instance == null) {
                // instance = new ParkingLot();
            }
            return instance;
        }
        ParkingLot(){
            
        }
    }

    public static void main(String[] args) {
        
    }

}
